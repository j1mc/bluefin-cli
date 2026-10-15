package main

import (
	"os"
	"strings"
	"testing"
)

func TestShellInstallersRequireChecksums(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"install.sh", "src/bluefin-cli/install.sh"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			script := string(data)
			for _, required := range []string{
				"checksums.txt",
				"expected_sha=",
				"${#expected_sha}",
				"sha256sum",
				"shasum -a 256",
				"actual_sha",
				"checksum verification failed",
			} {
				if !strings.Contains(script, required) {
					t.Errorf("installer does not enforce %q", required)
				}
			}
		})
	}
}
