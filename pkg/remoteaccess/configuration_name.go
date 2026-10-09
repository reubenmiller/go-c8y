package remoteaccess

import (
	"regexp"
	"slices"
	"strings"
)

// ConfigurationName is the meaning of a remote access configuration name following the convention
// `<scheme>[+<option>...]:<label>` (e.g. `http:Node-RED`, `https+mux:Router`), as used by
// https://github.com/Cumulocity-IoT/cumulocity-remote-access-cloud-http-proxy
type ConfigurationName struct {
	// Scheme is the URL scheme of the service, "http" or "https"
	Scheme string
	// Multiplex is set by the `mux` option: the device supports remote access multiplexing
	Multiplex bool
	// Label is the name without the scheme and options
	Label string
}

var configurationNamePattern = regexp.MustCompile(`^(https?)((?:\+[a-z0-9-]+)*):(.*)$`)

// ParseConfigurationName parses a remote access configuration name. It returns false if the name
// does not follow the convention.
func ParseConfigurationName(name string) (ConfigurationName, bool) {
	match := configurationNamePattern.FindStringSubmatch(name)
	if match == nil {
		return ConfigurationName{}, false
	}
	options := strings.Split(strings.TrimPrefix(match[2], "+"), "+")
	return ConfigurationName{
		Scheme:    match[1],
		Multiplex: slices.Contains(options, "mux"),
		Label:     match[3],
	}, true
}
