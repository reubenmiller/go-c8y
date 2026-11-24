package dataprep

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// File names inside a rule package.
const (
	ConfigFile = "data-prep.yaml"
	AIChatFile = "ai-chat.yaml"
	TestsDir   = "tests"

	// ExportedSmartFunctionFile is the file name the Data Preparation
	// application uses for the smart function in its exports.
	ExportedSmartFunctionFile = "smartFunction.js"
)

// TransportMQTT is the only transport the platform supports.
const TransportMQTT = "mqtt"

// Payload formats of test inputs.
const (
	PayloadFormatJSON   = "json"
	PayloadFormatText   = "text"
	PayloadFormatBase64 = "base64"
)

// Limits are the package limits the server enforces. They are tenant
// settings (dataprep/ruleImport.*), so a tenant can use other values.
type Limits struct {
	MaxEntries   int
	MaxFileSize  int64
	MaxTotalSize int64
}

// DefaultLimits are the documented default limits.
var DefaultLimits = Limits{
	MaxEntries:   100,
	MaxFileSize:  1 << 20,
	MaxTotalSize: 10 << 20,
}

// ErrInvalidPackage is wrapped by every error Validate returns.
var ErrInvalidPackage = errors.New("invalid rule package")

// Input is the "input" block of data-prep.yaml: which messages the rule
// receives.
type Input struct {
	// Transport is the transport type. Only "mqtt" is supported.
	Transport string `yaml:"transport" json:"transport"`
	// TopicPattern filters topics. * matches zero or more characters;
	// MQTT wildcards (+ and #) and ** are not supported.
	TopicPattern string `yaml:"topicPattern" json:"topicPattern"`
	// ClientIDPattern filters device client ids. * is a wildcard.
	ClientIDPattern string `yaml:"clientIDPattern,omitempty" json:"clientIDPattern,omitempty"`
	// TransportID identifies the transport connection. Defaults to
	// Transport.
	TransportID string `yaml:"transportID,omitempty" json:"transportID,omitempty"`
}

// RuleConfig is the content of data-prep.yaml.
type RuleConfig struct {
	// SmartFunctionFile is the name of the bundled JavaScript file at the
	// archive root.
	SmartFunctionFile string   `yaml:"smartFunctionFile"`
	Description       string   `yaml:"description,omitempty"`
	Tags              []string `yaml:"tags,omitempty"`
	// Disabled deploys the rule in the disabled state.
	Disabled bool  `yaml:"disabled,omitempty"`
	Input    Input `yaml:"input"`

	// Extra keeps data-prep.yaml fields this SDK does not know, so reading
	// and writing a package does not drop them.
	Extra map[string]any `yaml:",inline"`
}

// TestInput is one device message fed to the smart function.
type TestInput struct {
	Payload string `yaml:"payload" json:"payload"`
	// PayloadFormat is "json", "text" or "base64".
	PayloadFormat string `yaml:"payloadFormat" json:"payloadFormat"`
	Topic         string `yaml:"topic" json:"topic"`
	ClientID      string `yaml:"clientID" json:"clientID"`
	// Time is an ISO 8601 timestamp with a time zone.
	Time string `yaml:"time" json:"time"`
}

// TestCase is a named test: input messages and, optionally, the outputs
// they should produce.
type TestCase struct {
	Inputs []TestInput `yaml:"inputs" json:"inputs"`
	// ExpectedOutput lists the expected Cumulocity objects of all inputs,
	// in order. The server only stores it; CompareResults checks it.
	ExpectedOutput []map[string]any `yaml:"expectedOutput,omitempty" json:"expectedOutput,omitempty"`
}

// JSONInput returns a test input with a JSON payload, timestamped now.
func JSONInput(topic, clientID, payload string) TestInput {
	return TestInput{Payload: payload, PayloadFormat: PayloadFormatJSON, Topic: topic, ClientID: clientID, Time: now()}
}

// TextInput returns a test input with a text payload, timestamped now.
func TextInput(topic, clientID, payload string) TestInput {
	return TestInput{Payload: payload, PayloadFormat: PayloadFormatText, Topic: topic, ClientID: clientID, Time: now()}
}

// BinaryInput returns a test input with a binary payload (sent base64
// encoded), timestamped now.
func BinaryInput(topic, clientID string, payload []byte) TestInput {
	return TestInput{Payload: base64.StdEncoding.EncodeToString(payload), PayloadFormat: PayloadFormatBase64, Topic: topic, ClientID: clientID, Time: now()}
}

func now() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

// RulePackage is a rule in memory: its configuration, smart function and
// tests. Build one with NewPackage, PackageFromDir or PackageFromArchive,
// and turn it into the deployable tar.gz with Archive.
type RulePackage struct {
	Config        RuleConfig
	SmartFunction []byte
	Tests         map[string]TestCase
	// AIChat is the AI assistant chat history (ai-chat.yaml), kept as-is.
	// When nil, the server keeps the rule's existing history.
	AIChat []byte
	// Limits overrides DefaultLimits for Validate.
	Limits *Limits
}

// NewPackage creates a package from a configuration. Add the code with
// WithSmartFunction and tests with WithTest.
func NewPackage(cfg RuleConfig) *RulePackage {
	if cfg.SmartFunctionFile == "" {
		cfg.SmartFunctionFile = ExportedSmartFunctionFile
	}
	return &RulePackage{Config: cfg, Tests: map[string]TestCase{}}
}

// WithSmartFunction sets the bundled JavaScript code.
func (p *RulePackage) WithSmartFunction(code string) *RulePackage {
	p.SmartFunction = []byte(code)
	return p
}

// WithTest adds or replaces a test.
func (p *RulePackage) WithTest(name string, tc TestCase) *RulePackage {
	if p.Tests == nil {
		p.Tests = map[string]TestCase{}
	}
	p.Tests[name] = tc
	return p
}

// TestNames returns the test names, sorted.
func (p *RulePackage) TestNames() []string {
	return slices.Sorted(maps.Keys(p.Tests))
}

func (p *RulePackage) limits() Limits {
	if p.Limits != nil {
		return *p.Limits
	}
	return DefaultLimits
}

// Validate checks the package against the documented format and limits,
// so problems show up before anything is uploaded. Every error wraps
// ErrInvalidPackage.
func (p *RulePackage) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	c := p.Config
	switch {
	case c.SmartFunctionFile == "":
		add("%s: smartFunctionFile is required", ConfigFile)
	case !isRootFile(c.SmartFunctionFile):
		add("%s: smartFunctionFile %q must be a file name at the archive root", ConfigFile, c.SmartFunctionFile)
	case c.SmartFunctionFile == ConfigFile || c.SmartFunctionFile == AIChatFile:
		add("%s: smartFunctionFile %q clashes with a reserved file name", ConfigFile, c.SmartFunctionFile)
	}
	if len(p.SmartFunction) == 0 {
		add("smart function %q is empty", c.SmartFunctionFile)
	}
	if c.Input.Transport != TransportMQTT {
		add("%s: input.transport must be %q, got %q", ConfigFile, TransportMQTT, c.Input.Transport)
	}
	switch {
	case c.Input.TopicPattern == "":
		add("%s: input.topicPattern is required", ConfigFile)
	case strings.Contains(c.Input.TopicPattern, "**"):
		add("%s: input.topicPattern %q must not contain **", ConfigFile, c.Input.TopicPattern)
	case strings.ContainsAny(c.Input.TopicPattern, "+#"):
		add("%s: input.topicPattern %q uses MQTT wildcards (+ or #); use * instead", ConfigFile, c.Input.TopicPattern)
	}

	for _, name := range p.TestNames() {
		if name == "" {
			add("tests: a test has an empty name")
			continue
		}
		tc := p.Tests[name]
		if len(tc.Inputs) == 0 {
			add("test %q: needs at least one input", name)
		}
		for i, in := range tc.Inputs {
			if err := validateInput(in); err != nil {
				add("test %q input %d: %v", name, i, err)
			}
		}
	}

	// Size limits are checked on the actual archive entries.
	entries, err := p.entries()
	if err != nil {
		add("%v", err)
	} else {
		lim := p.limits()
		var total int64
		if lim.MaxEntries > 0 && len(entries) > lim.MaxEntries {
			add("package has %d entries, the limit is %d", len(entries), lim.MaxEntries)
		}
		for _, e := range entries {
			size := int64(len(e.data))
			total += size
			if lim.MaxFileSize > 0 && size > lim.MaxFileSize {
				add("%s is %d bytes, the limit is %d", e.name, size, lim.MaxFileSize)
			}
		}
		if lim.MaxTotalSize > 0 && total > lim.MaxTotalSize {
			add("package is %d bytes in total, the limit is %d", total, lim.MaxTotalSize)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrInvalidPackage, errors.Join(errs...))
}

func validateInput(in TestInput) error {
	var missing []string
	for field, v := range map[string]string{"payload": in.Payload, "topic": in.Topic, "clientID": in.ClientID, "time": in.Time} {
		if v == "" && field != "payload" {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	switch in.PayloadFormat {
	case PayloadFormatJSON, PayloadFormatText:
	case PayloadFormatBase64:
		if _, err := base64.StdEncoding.DecodeString(in.Payload); err != nil {
			return fmt.Errorf("payload is not valid base64: %w", err)
		}
	default:
		return fmt.Errorf("payloadFormat must be json, text or base64, got %q", in.PayloadFormat)
	}
	if _, err := time.Parse(time.RFC3339, in.Time); err != nil {
		return fmt.Errorf("time %q is not an ISO 8601 timestamp with a time zone", in.Time)
	}
	return nil
}

func isRootFile(name string) bool {
	return name != "" && name == path.Base(name) && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}

type entry struct {
	name string
	data []byte
}

// entries returns the archive entries in a stable order.
func (p *RulePackage) entries() ([]entry, error) {
	cfg, err := yaml.Marshal(p.Config)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", ConfigFile, err)
	}
	out := []entry{
		{ConfigFile, cfg},
		{p.Config.SmartFunctionFile, p.SmartFunction},
	}
	if p.AIChat != nil {
		out = append(out, entry{AIChatFile, p.AIChat})
	}
	seen := map[string]string{}
	for _, name := range p.TestNames() {
		file := TestsDir + "/" + EncodeTestName(name) + ".yaml"
		if other, dup := seen[file]; dup {
			return nil, fmt.Errorf("tests %q and %q map to the same file %s", other, name, file)
		}
		seen[file] = name
		b, err := yaml.Marshal(p.Tests[name])
		if err != nil {
			return nil, fmt.Errorf("encode test %q: %w", name, err)
		}
		out = append(out, entry{file, b})
	}
	return out, nil
}

// Entries lists the file names in the archive.
func (p *RulePackage) Entries() []string {
	entries, err := p.entries()
	if err != nil {
		return nil
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}

// Archive validates the package and returns it as the gzip-compressed TAR
// the deploy endpoint expects. The output is reproducible: the same
// package always gives the same bytes.
func (p *RulePackage) Archive() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	entries, err := p.entries()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.data)), ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(e.data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Checksum is the SHA-256 of the package contents (not of the archive
// bytes), so it only changes when the configuration, code or tests change.
func (p *RulePackage) Checksum() string {
	h := sha256.New()
	entries, _ := p.entries()
	for _, e := range entries {
		fmt.Fprintf(h, "%s\x00%d\x00", e.name, len(e.data))
		h.Write(e.data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// PackageFromDir reads a package from a folder laid out like the archive
// (data-prep.yaml, the smart function file and tests/*.yaml), for example
// the build output of the rules template repository.
func PackageFromDir(dir string) (*RulePackage, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != ConfigFile && rel != AIChatFile && !strings.HasPrefix(rel, TestsDir+"/") && strings.Contains(rel, "/") {
			return nil // nested files are ignored, as by the server
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return packageFromFiles(files)
}

// PackageFromArchive reads a package from a tar.gz, such as one exported by
// the Data Preparation application. Entries with ".." are rejected, like
// the server does.
func PackageFromArchive(r io.Reader) (*RulePackage, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("%w: not a gzip archive: %w", ErrInvalidPackage, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidPackage, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := strings.TrimPrefix(hdr.Name, "./")
		if slices.Contains(strings.Split(name, "/"), "..") {
			return nil, fmt.Errorf("%w: entry %q contains ..", ErrInvalidPackage, hdr.Name)
		}
		b, err := io.ReadAll(io.LimitReader(tr, DefaultLimits.MaxTotalSize+1))
		if err != nil {
			return nil, err
		}
		files[name] = b
	}
	return packageFromFiles(files)
}

func packageFromFiles(files map[string][]byte) (*RulePackage, error) {
	raw, ok := files[ConfigFile]
	if !ok {
		return nil, fmt.Errorf("%w: %s is missing", ErrInvalidPackage, ConfigFile)
	}
	var cfg RuleConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidPackage, ConfigFile, err)
	}
	p := &RulePackage{Config: cfg, Tests: map[string]TestCase{}, AIChat: files[AIChatFile]}
	code, ok := files[cfg.SmartFunctionFile]
	if !ok {
		return nil, fmt.Errorf("%w: smart function %q is missing", ErrInvalidPackage, cfg.SmartFunctionFile)
	}
	p.SmartFunction = code

	for _, name := range slices.Sorted(maps.Keys(files)) {
		base, ok := strings.CutPrefix(name, TestsDir+"/")
		if !ok || strings.Contains(base, "/") || !strings.HasSuffix(base, ".yaml") {
			continue
		}
		testName := DecodeTestName(strings.TrimSuffix(base, ".yaml"))
		if _, dup := p.Tests[testName]; dup {
			return nil, fmt.Errorf("%w: two test files define test %q", ErrInvalidPackage, testName)
		}
		var tc TestCase
		if err := yaml.Unmarshal(files[name], &tc); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrInvalidPackage, name, err)
		}
		p.Tests[testName] = tc
	}
	return p, nil
}

// WriteDir writes the package to dir in the archive layout. Existing files
// with the same names are overwritten; other files are left alone.
func (p *RulePackage) WriteDir(dir string) error {
	entries, err := p.entries()
	if err != nil {
		return err
	}
	for _, e := range entries {
		target := filepath.Join(dir, filepath.FromSlash(e.name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, e.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// EncodeTestName turns a test name into a file name the way the Data
// Preparation application does: %, control characters, a leading ".", and
// / \ : * ? " < > | are written as %XX (UTF-8 bytes, as in a URL).
func EncodeTestName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c < 0x20 || c == 0x7f || strings.IndexByte(`%/\:*?"<>|`, c) >= 0 || (i == 0 && c == '.') {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// DecodeTestName reverses EncodeTestName. A % that is not followed by two
// hex digits is kept as it is.
func DecodeTestName(file string) string {
	var b []byte
	for i := 0; i < len(file); i++ {
		if file[i] == '%' && i+2 < len(file) {
			if v, err := hex.DecodeString(file[i+1 : i+3]); err == nil {
				b = append(b, v[0])
				i += 2
				continue
			}
		}
		b = append(b, file[i])
	}
	return string(b)
}
