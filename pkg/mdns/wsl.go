//go:build linux

package mdns

import (
	"os"
	"strings"
)

// isWSL2 reports whether the current process is running inside WSL2 by
// inspecting the kernel version string in /proc/version.
func isWSL2() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	// WSL2 kernel strings contain both "microsoft" and "wsl".
	// Example: "Linux version 5.15.90.1-microsoft-standard-WSL2 ..."
	return strings.Contains(lower, "microsoft") && strings.Contains(lower, "wsl")
}
