// Package launcher prepares the isolated GH_CONFIG_DIR and GIT_CONFIG_GLOBAL
// files and execs the real claude binary with full signal forwarding and
// deferred cleanup.
package launcher

import (
	"fmt"
	"os"
)

// TempGHConfigDir creates an empty 0700 directory suitable for use as
// GH_CONFIG_DIR. Returns the directory path and a cleanup func.
//
// An empty GH_CONFIG_DIR is sufficient because GH_TOKEN takes precedence over
// any hosts.yml; gh will materialise state.yml / config.yml inside the dir on
// demand without needing it pre-seeded.
func TempGHConfigDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "claude-github-app-*-gh")
	if err != nil {
		return "", nil, fmt.Errorf("create GH_CONFIG_DIR: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("chmod GH_CONFIG_DIR: %w", err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
