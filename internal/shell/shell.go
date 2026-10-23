package shell

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed resources/shell.sh
var shellShScript string

//go:embed resources/shell.fish
var shellFishScript string

//go:embed resources/shell.ps1
var shellPowerShellScript string

const shellMaker = "# bluefin-cli shell-config"
const blingMarker = "# bluefin-cli bling"

func Toggle(shell string, enable bool) error {
	if isPowerShellShell(shell) {
		return togglePowerShell(enable)
	}

	var configFile string
	var rcLine string

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	switch shell {
	case "bash":
		configFile = filepath.Join(home, ".bashrc")
		rcLine = fmt.Sprintf(`eval "$(bluefin-cli init bash)" %s`, shellMaker)
	case "zsh":
		configFile = filepath.Join(home, ".zshrc")
		rcLine = fmt.Sprintf(`eval "$(bluefin-cli init zsh)" %s`, shellMaker)
	case "fish":
		configFile = filepath.Join(home, ".config/fish/config.fish")
		rcLine = fmt.Sprintf(`bluefin-cli init fish | source %s`, shellMaker)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) && enable {
			// Create if doesn't exist and we are enabling
			// For fish, ensure dir exists
			if shell == "fish" {
				if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
					return err
				}
			}
			content = []byte("")
		} else if os.IsNotExist(err) && !enable {
			printInfo("%s is already disabled for %s", shell, shell)
			return nil
		} else {
			return err
		}
	}

	text := string(content)
	hasLine := strings.Contains(text, shellMaker)

	if enable {
		if hasLine {
			printInfo("%s is already enabled for %s", shell, shell)
			if cfg, err := LoadConfig(shell); err == nil {
				InstallTools(shell, cfg)
			}
			return nil
		}

		f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()

		prefix := "\n"
		if len(text) == 0 || strings.HasSuffix(text, "\n") {
			prefix = ""
		}

		if _, err := f.WriteString(prefix + rcLine + "\n"); err != nil {
			return err
		}
		printSuccess("✓ Enabled shell experience for %s", shell)
	} else {
		if !hasLine {
			printInfo("%s is already disabled for %s", shell, shell)
			return nil
		}

		// Remove the lines containing the marker
		lines := strings.Split(text, "\n")
		var newLines []string
		for _, line := range lines {
			if !strings.Contains(line, shellMaker) && !strings.Contains(line, blingMarker) {
				newLines = append(newLines, line)
			}
		}

		output := strings.Join(newLines, "\n")
		// Trim extra newlines at the end
		output = strings.TrimRight(output, "\n") + "\n"

		if err := os.WriteFile(configFile, []byte(output), 0644); err != nil {
			return err
		}
		printSuccess("✓ Disabled shell experience for %s", shell)
	}

	if enable {
		if cfg, err := LoadConfig(shell); err == nil {
			InstallTools(shell, cfg)
		}
	}

	return nil
}

// initCacheMtime returns the modification time of the shell config file
// (used to decide whether the cached init output is still valid).
func initCacheMtime(shell string) (time.Time, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(configPath)
	if err != nil {
		// Config doesn't exist yet — treat as epoch so any cache is
		// considered newer and we skip the expensive tool checks.
		return time.Time{}, nil
	}
	return info.ModTime(), nil
}

// loadInitCache returns the cached init script for shell if it's still fresh
// (binary and config haven't changed since the cache was written).
func loadInitCache(shell string) (string, bool) {
	cachePath, err := getInitCachePath(shell)
	if err != nil {
		return "", false
	}
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return "", false
	}
	cacheMtime := cacheInfo.ModTime()

	// Must be newer than the bluefin-cli binary.
	if exe, err := os.Executable(); err == nil {
		if exeInfo, err := os.Stat(exe); err == nil {
			if !cacheMtime.After(exeInfo.ModTime()) {
				return "", false
			}
		}
	}

	// Must be newer than the shell config file.
	if cfgMtime, err := initCacheMtime(shell); err == nil {
		if !cacheMtime.After(cfgMtime) {
			return "", false
		}
	}

	content, err := os.ReadFile(cachePath)
	if err != nil {
		return "", false
	}
	return string(content), true
}

// saveInitCache writes the generated init script to a cache file.
func saveInitCache(shell, script string) {
	cachePath, err := getInitCachePath(shell)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return
	}
	_ = os.WriteFile(cachePath, []byte(script), 0644)
}

// getInitCachePath returns the path to the cached init script for a shell.
func getInitCachePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "bluefin-cli", fmt.Sprintf("init-%s.sh", shell)), nil
}

func Init(shell string, config *Config) (string, error) {
	if config == nil {
		config = DefaultConfig(shell)
	}

	// Try to serve a cached init script to avoid expensive tool-availability
	// checks on every interactive shell startup (issue #59). The cache is
	// invalidated when the bluefin-cli binary or the shell config changes.
	if cached, ok := loadInitCache(shell); ok {
		return cached, nil
	}

	tools := ToolsForShell(shell)
	configUpdated := false

	// Synchronize configuration with installed tools in PATH
	for _, tool := range tools {
		isInstalled := isBinaryAvailable(tool)

		isEnabled := config.IsEnabled(tool.Name)

		if isEnabled && !isInstalled {
			fmt.Fprintf(os.Stderr, "bluefin-cli: %s is enabled but '%s' was not found in PATH.\n", tool.Name, tool.Binary)
			fmt.Fprintf(os.Stderr, "  - Install it: brew install %s\n", tool.GetBrewPkg())
			fmt.Fprintf(os.Stderr, "  - Or disable it: bluefin-cli shell config\n")

			config.SetEnabled(tool.Name, false)
			configUpdated = true

			// If Glow is disabled, also disable MOTD
			if tool.Name == "Glow" {
				config.SetEnabled("Motd", false)
			}
		} else if !isEnabled && isInstalled {
			// Auto-enable if found and previously disabled
			config.SetEnabled(tool.Name, true)
			configUpdated = true
			fmt.Fprintf(os.Stderr, "bluefin-cli: %s found in PATH, automatically enabling.\n", tool.Name)

			// If Glow is found, also enable MOTD
			if tool.Name == "Glow" {
				config.SetEnabled("Motd", true)
			}
		}
	}

	// MOTD consistency check (if manual override happened)
	if config.IsEnabled("Motd") && !config.IsEnabled("Glow") {
		// If MOTD enabled but Glow disabled, try to enable Glow
		if _, err := exec.LookPath("glow"); err == nil {
			config.SetEnabled("Glow", true)
			configUpdated = true
		} else {
			// Glow missing, must disable MOTD
			config.SetEnabled("Motd", false)
			configUpdated = true
		}
	}

	if configUpdated {
		if err := SaveConfig(config); err != nil {
			fmt.Fprintf(os.Stderr, "bluefin-cli: failed to save updated config: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "bluefin-cli: configuration synchronized with installed components.\n")
		}
	}

	var sb strings.Builder

	if isPowerShellShell(shell) {
		for _, tool := range tools {
			enabled := config.IsEnabled(tool.Name)
			fmt.Fprintf(&sb, "$env:%s = \"%d\"\n", tool.GetEnvVar(), boolToInt(enabled))
		}
		fmt.Fprintf(&sb, "$env:BLUEFIN_SHELL_ENABLE_MOTD = \"%d\"\n", boolToInt(config.IsEnabled("Motd")))

		sb.WriteString("\n")
		sb.WriteString(shellPowerShellScript)
		result := sb.String()
		saveInitCache(shell, result)
		return result, nil
	}

	for _, tool := range tools {
		enabled := config.IsEnabled(tool.Name)

		if shell == "fish" {
			fmt.Fprintf(&sb, "set -gx %s %d\n", tool.GetEnvVar(), boolToInt(enabled))
		} else {
			fmt.Fprintf(&sb, "export %s=%d\n", tool.GetEnvVar(), boolToInt(enabled))
		}
	}

	if shell == "fish" {
		fmt.Fprintf(&sb, "set -gx BLUEFIN_SHELL_ENABLE_MOTD %d\n", boolToInt(config.IsEnabled("Motd")))
		fmt.Fprintf(&sb, "set -gx BLING_SHELL %s\n", shell)
	} else {
		fmt.Fprintf(&sb, "export BLUEFIN_SHELL_ENABLE_MOTD=%d\n", boolToInt(config.IsEnabled("Motd")))
		fmt.Fprintf(&sb, "export BLING_SHELL=\"%s\"\n", shell)
	}

	sb.WriteString("\n")

	if shell == "fish" {
		sb.WriteString(shellFishScript)
	} else {
		sb.WriteString(shellShScript)
	}

	result := sb.String()
	saveInitCache(shell, result)
	return result, nil
}
