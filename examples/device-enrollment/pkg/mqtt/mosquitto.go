package mqtt

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildMosquittoSubArgs(opts MqttClientOptions) []string {
	mosquitoArgs := make([]string, 0, 15)

	if len(opts.Payload) > 0 {
		mosquitoArgs = append(mosquitoArgs, "mosquitto_pub")
	} else {
		mosquitoArgs = append(mosquitoArgs, "mosquitto_sub")
	}

	if opts.ClientID != "" {
		mosquitoArgs = append(mosquitoArgs, "-i", opts.ClientID)
	}

	if opts.KeyFile != "" {
		mosquitoArgs = append(mosquitoArgs, "--key", opts.KeyFile)
	}

	if opts.CertFile != "" {
		mosquitoArgs = append(mosquitoArgs, "--cert", opts.CertFile)
	}

	// Set Root ca
	// Set some sensible defaults
	defaultRootCAs := []string{}
	if opts.CAPath != "" {
		defaultRootCAs = append(defaultRootCAs, opts.CAPath)
	}
	if len(defaultRootCAs) == 0 {
		if homebrewPrefix, err := exec.Command("brew", "--prefix").Output(); err == nil {
			defaultRootCAs = append(defaultRootCAs, filepath.Join(string(bytes.TrimSpace(homebrewPrefix)), "etc/ca-certificates/cert.pem"))
		}
	}

	for _, ca := range defaultRootCAs {
		if stat, err := os.Stat(ca); err == nil {
			if stat.IsDir() {
				mosquitoArgs = append(mosquitoArgs, "--capath", ca)
			} else {
				mosquitoArgs = append(mosquitoArgs, "--cafile", ca)
			}
			break
		}
	}

	if opts.Username != "" {
		mosquitoArgs = append(mosquitoArgs, "-u", opts.Username)
	}

	if opts.Host != "" {
		mosquitoArgs = append(mosquitoArgs, "-h", opts.Host)
	}

	if opts.Port != 0 {
		mosquitoArgs = append(mosquitoArgs, "-p", fmt.Sprintf("%d", opts.Port))
	}

	mosquitoArgs = append(mosquitoArgs, "--debug", "-t", opts.Topic)

	return mosquitoArgs
}
