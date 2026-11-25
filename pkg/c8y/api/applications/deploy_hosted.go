package applications

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core/artifact" //nolint:misspell
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/zipUtilities"
)

// HostedManifestFile is the name of the optional manifest file in the root of a
// hosted web application.
const HostedManifestFile = "cumulocity.json"

// HostedManifest is the subset of a hosted web application's manifest
// (cumulocity.json) that DeployHosted reads.
type HostedManifest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Key         string `json:"key,omitempty"`
	Version     string `json:"version,omitempty"`
	ContextPath string `json:"contextPath,omitempty"`
}

// DeployHostedOptions controls how DeployHosted creates or updates a hosted web
// application.
type DeployHostedOptions struct {
	// Path to the web application: either a folder (its contents are zipped,
	// so index.html must be in the folder's root) or an existing .zip file
	// (uploaded as is). It may be empty only when SkipUpload is set.
	Path string

	// Name of the application. Defaults to the name in the manifest
	// (cumulocity.json), then to the file or folder name without any version
	// suffix (e.g. "myapp-1.0.0.zip" -> "myapp").
	Name string

	// Key of the application. Defaults to the key in the manifest, then to
	// "<name>-application-key".
	Key string

	// ContextPath of the application. Defaults to the context path in the
	// manifest, then to the name.
	ContextPath string

	// Availability of the application for other tenants (MARKET or PRIVATE).
	// Leave empty to use the platform default (PRIVATE).
	Availability string

	// ResourcesURL of the application. Defaults to "/".
	ResourcesURL string

	// Owner is the tenant ID which owns the application. When set, only
	// applications owned by this tenant are considered when looking for an
	// existing application, so a subscribed application with the same name
	// owned by a parent tenant is never updated.
	Owner string

	// SkipUpload only creates the application (or gets the existing one)
	// without uploading a binary.
	SkipUpload bool

	// SkipActivation uploads the binary but does not activate it, so the
	// currently active version (if any) stays in use.
	SkipActivation bool

	// KeepBinaries is the number of binaries (web application versions) to
	// keep after a successful upload. The oldest binaries (by creation time)
	// beyond this number are deleted, but the active binary and the binary
	// which was just uploaded are always kept. Zero or less keeps all
	// binaries.
	KeepBinaries int
}

// DeployHosted creates a hosted (web) application, or updates an existing one
// with the same name, from a local folder or zip file:
//
//  1. Look for an existing HOSTED application with the name (owned by
//     opt.Owner when set); create it when none exists.
//  2. Zip the folder (a .zip file is used as is) and upload it as a new
//     binary of the application.
//  3. Activate the uploaded binary.
//
// Existing application properties are not changed, only a new binary is
// uploaded and activated. The result's Status is StatusCreated for a new
// application, StatusUpdated when a binary was uploaded to an existing one and
// StatusOK when nothing was changed. The result's Meta contains "applicationId",
// "binaryId" (when uploaded), "created", "activated" and, when KeepBinaries is
// set, "deletedBinaries" (the ids of the binaries removed by the rotation).
//
// In dry-run mode the lookup is still sent so the result shows whether the
// application would be created, but the upload and activation are skipped.
func (s *Service) DeployHosted(ctx context.Context, opt DeployHostedOptions) op.Result[jsonmodels.Application] {
	return op.ExecuteOrDefer(ctx, op.WithMeta(op.WithExecutor(op.Result[jsonmodels.Application]{}, func(execCtx context.Context) op.Result[jsonmodels.Application] {
		return s.deployHosted(execCtx, opt)
	}), "operation", "deployHosted"))
}

func (s *Service) deployHosted(ctx context.Context, opt DeployHostedOptions) op.Result[jsonmodels.Application] {
	skipUpload := opt.SkipUpload || opt.Path == ""
	if !opt.SkipUpload && opt.Path == "" {
		return op.Failed[jsonmodels.Application](errors.New("a path to a folder or zip file is required unless SkipUpload is set"), false)
	}
	if opt.Path != "" {
		if _, err := os.Stat(opt.Path); err != nil {
			return op.Failed[jsonmodels.Application](fmt.Errorf("could not read web application. path=%s. %w", opt.Path, err), false)
		}
	}

	details, err := hostedApplicationDetails(opt)
	if err != nil {
		return op.Failed[jsonmodels.Application](err, false)
	}

	// Find or create the application
	appResult, created := s.findOrCreateHosted(ctx, details, opt.Owner)
	if appResult.Err != nil || ctxhelpers.IsDryRun(ctx) {
		return withDeployMeta(appResult, created, "", false)
	}
	app := appResult.Data

	if skipUpload {
		return withDeployMeta(appResult, created, "", false)
	}

	// Upload binary
	zipFile, cleanup, err := packageHostedApplication(opt.Path)
	if err != nil {
		return withDeployMeta(op.Failed[jsonmodels.Application](err, false), created, "", false)
	}
	defer cleanup()

	uploadResult := s.Upload(ctx, app.ID(), UploadFileOptions{
		FilePath:    zipFile,
		Name:        filepath.Base(zipFile),
		ContentType: "application/zip",
	})
	if uploadResult.Err != nil {
		uploadResult.Err = fmt.Errorf("failed to upload web application binary. path=%s. %w", zipFile, uploadResult.Err)
		return withDeployMeta(uploadResult, created, "", false)
	}
	binaryID := uploadResult.Data.ID()

	result := appResult
	if !opt.SkipActivation {
		if binaryID == "" {
			return withDeployMeta(op.Failed[jsonmodels.Application](errors.New("failed to activate the new application version because the binary id is empty"), false), created, "", false)
		}
		result = s.Update(ctx, app.ID(), &model.Application{ActiveVersionID: binaryID}).IgnoreConflict()
		if result.Err != nil {
			result.Err = fmt.Errorf("failed to activate application binary. binaryId=%s. %w", binaryID, result.Err)
			return withDeployMeta(result, created, binaryID, false)
		}
		if result.Status == op.StatusDuplicate {
			// Already active (409): keep the application from the lookup
			result.Data = app
		}
	}

	if created {
		result.Status = op.StatusCreated
	} else {
		result.Status = op.StatusUpdated
	}
	result = withDeployMeta(result, created, binaryID, !opt.SkipActivation)

	if opt.KeepBinaries > 0 {
		activeID := app.Get("activeVersionId").String()
		if !opt.SkipActivation {
			activeID = binaryID
		}
		rotation := s.rotateBinaries(ctx, app.ID(), RotateBinariesOptions{
			Keep:            opt.KeepBinaries,
			ActiveVersionID: activeID,
			Protect:         []string{binaryID},
		})
		result = op.WithMeta(result, "deletedBinaries", deletedBinaries(rotation))
		if rotation.Err != nil {
			result.Err = fmt.Errorf("deployed the web application but failed to delete old binaries. %w", rotation.Err)
			result.Status = op.StatusPartial
		}
	}
	return result
}

func withDeployMeta(r op.Result[jsonmodels.Application], created bool, binaryID string, activated bool) op.Result[jsonmodels.Application] {
	r = op.WithMeta(op.WithMeta(op.WithMeta(r, "operation", "deployHosted"), "created", created), "activated", activated)
	if id := r.Data.ID(); id != "" {
		r = op.WithMeta(r, "applicationId", id)
	}
	if binaryID != "" {
		r = op.WithMeta(r, "binaryId", binaryID)
	}
	return r
}

// findOrCreateHosted returns the existing hosted application with the given
// name, or creates it. The boolean is true when the application was created.
func (s *Service) findOrCreateHosted(ctx context.Context, details *model.Application, owner string) (op.Result[jsonmodels.Application], bool) {
	// Always look up for real, even in dry-run mode, so the dry run shows
	// whether the application would be created or updated.
	lookupCtx := ctxhelpers.ResolutionContext(ctx)
	existing := s.List(lookupCtx, ListOptions{
		Name:  details.Name,
		Type:  TypeHosted,
		Owner: owner,
	})
	if existing.Err != nil {
		existing.Err = fmt.Errorf("failed to find hosted application. name=%s. %w", details.Name, existing.Err)
		return existing, false
	}
	for item := range existing.Data.Iter() {
		app := jsonmodels.NewApplication(item.Bytes())
		// The platform matches the name exactly, but guard against a server
		// which ignores the filter.
		if app.Name() != details.Name || !strings.EqualFold(app.Type(), TypeHosted) {
			continue
		}
		if owner != "" && app.Get("owner.tenant.id").String() != owner {
			continue
		}
		return s.Get(ctx, app.ID()), false
	}

	created := s.Create(ctx, details)
	if created.Err != nil {
		created.Err = fmt.Errorf("failed to create hosted application. name=%s. %w", details.Name, created.Err)
	}
	return created, true
}

// hostedApplicationDetails builds the application body from the options and
// the manifest in the folder or zip file (if any).
func hostedApplicationDetails(opt DeployHostedOptions) (*model.Application, error) {
	manifest := HostedManifest{}
	if opt.Path != "" {
		// A missing or invalid manifest is not an error: it is optional
		_ = ReadHostedManifest(opt.Path, &manifest)
	}

	app := &model.Application{
		Type:         TypeHosted,
		Availability: opt.Availability,
		ResourcesURL: "/",
	}

	app.Name = firstNonEmpty(opt.Name, manifest.Name)
	if app.Name == "" && opt.Path != "" {
		app.Name = artifact.ParseName(filepath.Clean(opt.Path)) //nolint:misspell
	}
	if app.Name == "" {
		return nil, errors.New("could not detect the application name. Set the name explicitly")
	}

	app.Key = firstNonEmpty(opt.Key, manifest.Key, app.Name+"-application-key")
	app.ContextPath = firstNonEmpty(opt.ContextPath, manifest.ContextPath, app.Name)
	app.ResourcesURL = firstNonEmpty(opt.ResourcesURL, "/")
	return app, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// ReadHostedManifest reads the hosted application manifest (cumulocity.json)
// from the root of a folder or a zip file into dst.
func ReadHostedManifest(path string, dst any) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		contents, err := os.ReadFile(filepath.Join(path, HostedManifestFile))
		if err != nil {
			return err
		}
		return json.Unmarshal(contents, dst)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if !strings.EqualFold(strings.TrimPrefix(file.Name, "/"), HostedManifestFile) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		contents, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		return json.Unmarshal(contents, dst)
	}
	return fmt.Errorf("%s not found in %s", HostedManifestFile, path)
}

// packageHostedApplication returns the path to a zip file of the web
// application. A file is returned as is; a folder's contents are zipped into a
// temporary file which is removed by cleanup.
func packageHostedApplication(src string) (zipFile string, cleanup func(), err error) {
	cleanup = func() {}
	info, err := os.Stat(src)
	if err != nil {
		return "", cleanup, err
	}
	if !info.IsDir() {
		return src, cleanup, nil
	}

	dir, err := os.MkdirTemp("", "c8y-hosted-app")
	if err != nil {
		return "", cleanup, fmt.Errorf("failed to create temp folder. %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	src = filepath.Clean(src)
	zipFile = filepath.Join(dir, filepath.Base(src)+".zip")
	// The trailing separator keeps the entry names relative to the folder
	// (e.g. "index.html" rather than "/index.html")
	if err := zipUtilities.ZipDirectoryContents(src+string(os.PathSeparator), zipFile); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("failed to zip web application folder. path=%s. %w", src, err)
	}
	return zipFile, cleanup, nil
}
