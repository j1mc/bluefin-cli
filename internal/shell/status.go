package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func toolsForCurrentPlatform() []Tool {
	if runtime.GOOS == "windows" {
		return ToolsForShell("powershell")
	}

	// For non-Windows platforms, filter out tools that are explicitly unsupported on common shells.
	// This ensures tools like gsudo (Windows-only) are not included.
	filtered := make([]Tool, 0, len(Tools))
	for _, tool := range Tools {
		if tool.SupportsShell("bash") || tool.SupportsShell("zsh") || tool.SupportsShell("fish") {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

func CheckStatus() map[string]bool {
	status := make(map[string]bool)
	shells := []string{"bash", "zsh", "fish"}
	home, _ := os.UserHomeDir()

	for _, shell := range shells {
		var configFile string
		switch shell {
		case "bash":
			configFile = filepath.Join(home, ".bashrc")
		case "zsh":
			configFile = filepath.Join(home, ".zshrc")
		case "fish":
			configFile = filepath.Join(home, ".config/fish/config.fish")
		}

		content, err := os.ReadFile(configFile)
		if err != nil {
			status[shell] = false
			continue
		}

		status[shell] = strings.Contains(string(content), shellMaker) || strings.Contains(string(content), "# bluefin-cli bling")
	}

	status["powershell"] = checkPowerShellStatus()
	status["pwsh"] = status["powershell"]

	return status
}

// homebrewPrefix resolves the Homebrew prefix using the same logic as the
// embedded shell.sh: respect HOMEBREW_PREFIX env var, probe common paths,
// and fall back to the Linuxbrew default. Returns "" when brew is absent.
func homebrewPrefix() string {
	if p := os.Getenv("HOMEBREW_PREFIX"); p != "" {
		return p
	}
	for _, p := range []string{"/opt/homebrew", "/usr/local", "/home/linuxbrew/.linuxbrew"} {
		if _, err := os.Stat(filepath.Join(p, "bin", "brew")); err == nil {
			return p
		}
	}
	return ""
}

// isBinaryAvailable reports whether tool.Binary can be found on PATH or,
// for uutils tools that Homebrew installs into a non-standard libexec
// directory, inside the brew opt prefix.
func isBinaryAvailable(tool Tool) bool {
	if _, err := exec.LookPath(tool.Binary); err == nil {
		return true
	}
	prefix := homebrewPrefix()
	if prefix == "" {
		return false
	}
	// uutils-coreutils / uutils-findutils / uutils-diffutils all follow
	// the same Homebrew layout: <prefix>/opt/<pkg>/libexec/uubin/<binary>
	optBin := filepath.Join(prefix, "opt", tool.GetBrewPkg(), "libexec", "uubin", tool.Binary)
	if _, err := os.Stat(optBin); err == nil {
		return true
	}
	return false
}

func CheckDependencies() map[string]bool {
	status := make(map[string]bool)

	for _, tool := range toolsForCurrentPlatform() {
		status[tool.Binary] = isBinaryAvailable(tool)
	}

	return status
}

// GetInstalledShells returns a list of shells that are available in the PATH
func GetInstalledShells() []string {
	var installed []string
	shells := []string{"bash", "zsh", "fish"}

	for _, s := range shells {
		if _, err := exec.LookPath(s); err == nil {
			installed = append(installed, s)
		}
	}

	if _, err := exec.LookPath("pwsh"); err == nil {
		installed = append(installed, "pwsh")
	} else if _, err := exec.LookPath("powershell"); err == nil {
		installed = append(installed, "powershell")
	} else if _, err := exec.LookPath("powershell.exe"); err == nil {
		installed = append(installed, "powershell")
	}

	return installed
}
