package shell

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/huh/v2"
	"github.com/tuna-os/bluefin-cli/internal/env"
)

// InstallTools iterates through the config and installs enabled tools.
func InstallTools(shell string, cfg *Config) {
	// If MOTD is enabled, ensure Glow is also considered enabled for installation
	if cfg.IsEnabled("Motd") {
		cfg.SetEnabled("Glow", true)
	}

	tools := ToolsForShell(shell)

	// First check if we need to install anything
	needsInstall := false
	for _, tool := range tools {
		if cfg.IsEnabled(tool.Name) {
			if !isBinaryAvailable(tool) {
				needsInstall = true
				break
			}
		}
	}

	if !needsInstall {
		return
	}

	if runtime.GOOS == "windows" {
		installToolsWindows(cfg)
		return
	}

	if env.IsAlpine() {
		installToolsAlpine(tools, cfg)
		return
	}

	// Ensure Homebrew is available
	if err := ensureHomebrew(); err != nil {
		printError("Skipping tool installation: %v", err)
		return
	}

	for _, tool := range tools {
		if cfg.IsEnabled(tool.Name) {
			if err := ensureTool(tool.Binary, tool.GetBrewPkg()); err != nil {
				printError("Warning: Failed to install %s: %v", tool.GetBrewPkg(), err)
			}
		}
	}
}

func installToolsWindows(cfg *Config) {
	if err := ensurePowerShellModules(); err != nil {
		printError("Warning: Failed to ensure PowerShell modules: %v", err)
	}

	availableManagers := availableWindowsManagers()
	if len(availableManagers) == 0 {
		printError("Skipping tool installation: winget not found")
		return
	}

	printInfo("Installing enabled components using winget.")

	tools := ToolsForShell("powershell")

	for _, tool := range tools {
		if strings.EqualFold(tool.Name, "Gsudo") && cfg.IsEnabled(tool.Name) {
			if err := ensureWindowsTool(tool.Binary, tool.Pkg, availableManagers); err != nil {
				printError("Warning: Failed to install %s: %v", tool.Pkg, err)
			}
			break
		}
	}

	if err := primeGsudoCache(); err != nil {
		printInfo("Proceeding without gsudo elevation cache: %s", err)
	}

	for _, tool := range tools {
		if strings.EqualFold(tool.Name, "Gsudo") {
			continue
		}

		if cfg.IsEnabled(tool.Name) {
			if err := ensureWindowsTool(tool.Binary, tool.Pkg, availableManagers); err != nil {
				printError("Warning: Failed to install %s: %v", tool.Pkg, err)
			}
		}
	}

	if cfg.IsEnabled("Motd") {
		if err := ensureWindowsTool("glow", "charmbracelet.glow", availableManagers); err != nil {
			printError("Warning: Failed to install glow: %v", err)
		}
	}
}

func ensurePowerShellModules() error {
	modules := []string{"PSReadLine"}

	var failures []string
	for _, moduleName := range modules {
		if err := ensurePowerShellModule(moduleName); err != nil {
			failures = append(failures, fmt.Sprintf("%s (%v)", moduleName, err))
		}
	}

	if err := ensurePSFileIcons(); err != nil {
		failures = append(failures, fmt.Sprintf("PSFileIcons (%v)", err))
	}

	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, ", "))
	}

	return nil
}

func ensurePowerShellModule(moduleName string) error {
	powerShellExe := windowsPowerShellExe()
	modulePathReset := `$env:PSModulePath = @("$env:USERPROFILE\Documents\WindowsPowerShell\Modules", "$env:ProgramFiles\WindowsPowerShell\Modules", "$env:WINDIR\System32\WindowsPowerShell\v1.0\Modules") -join ';'`

	checkScript := fmt.Sprintf("%s; if (Get-Module -ListAvailable -Name '%s' -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }", modulePathReset, moduleName)
	check := exec.Command(powerShellExe, "-NoProfile", "-NonInteractive", "-Command", checkScript)
	if err := check.Run(); err == nil {
		return nil
	}

	printInfo("⬇️  Installing PowerShell module %s...", moduleName)
	installScript := fmt.Sprintf("%s; $ErrorActionPreference='Stop'; Import-Module PackageManagement -ErrorAction Stop; Import-Module PowerShellGet -ErrorAction Stop; if (-not (Get-PackageProvider -Name NuGet -ListAvailable -ErrorAction SilentlyContinue)) { Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Scope CurrentUser -Force | Out-Null }; Set-PSRepository -Name PSGallery -InstallationPolicy Trusted -ErrorAction SilentlyContinue; Install-Module -Name '%s' -Scope CurrentUser -Repository PSGallery -Force -AllowClobber", modulePathReset, moduleName)
	install := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", installScript)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("%w (try in Windows PowerShell: Install-Module -Name %s -Scope CurrentUser -Repository PSGallery -Force)", err, moduleName)
	}

	verify := exec.Command(powerShellExe, "-NoProfile", "-NonInteractive", "-Command", checkScript)
	if err := verify.Run(); err != nil {
		return fmt.Errorf("module %s install completed but module is still not discoverable", moduleName)
	}

	printSuccess("✓ PowerShell module %s installed", moduleName)
	return nil
}

func windowsPowerShellExe() string {
	if runtime.GOOS == "windows" {
		systemPath := `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
		if _, err := os.Stat(systemPath); err == nil {
			return systemPath
		}
	}

	return "powershell.exe"
}

func ensureHomebrew() error {
	if _, err := exec.LookPath("brew"); err == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		return fmt.Errorf("homebrew not found on Windows; install shell tools with winget")
	}

	commonPaths := []string{"/home/linuxbrew/.linuxbrew/bin/brew", "/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			path := os.Getenv("PATH")
			if err := os.Setenv("PATH", path+string(os.PathListSeparator)+filepath.Dir(p)); err != nil {
				return fmt.Errorf("failed to update PATH: %w", err)
			}
			return nil
		}
	}

	printInfo("Homebrew is missing. It is required to install enabled components.")
	var install bool
	err := huh.NewConfirm().
		Title("Would you like to install Homebrew?").
		Value(&install).
		Run()
	if err != nil {
		return err
	}

	if !install {
		return fmt.Errorf("homebrew installation declined")
	}

	printInfo("⬇️  Installing Homebrew...")

	cmd := exec.Command("/bin/bash", "-c", "curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install homebrew: %w", err)
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			path := os.Getenv("PATH")
			if err := os.Setenv("PATH", path+string(os.PathListSeparator)+filepath.Dir(p)); err != nil {
				return fmt.Errorf("failed to update PATH: %w", err)
			}
			printSuccess("✓ Homebrew installed and added to PATH for this session.")
			return nil
		}
	}

	return fmt.Errorf("homebrew installed but not found in expected locations")
}

// EnsureInstalled installs a single tool (matched by Binary in Tools) via
// the best available manager on this platform, if it isn't already on
// PATH. Used by `doctor --fix`, which needs a one-off install outside the
// InstallTools bulk flow.
func EnsureInstalled(binary string) error {
	if _, err := exec.LookPath(binary); err == nil {
		return nil
	}
	var tool *Tool
	for i := range Tools {
		if Tools[i].Binary == binary {
			tool = &Tools[i]
			break
		}
	}
	if tool == nil {
		return fmt.Errorf("unknown tool %q", binary)
	}
	if runtime.GOOS == "windows" {
		return fmt.Errorf("install %s from the Shell menu on Windows", binary)
	}
	if env.IsAlpine() {
		mgr := alpinePackageManager()
		if mgr == "" {
			return fmt.Errorf("neither coldbrew nor apk found")
		}
		return installAlpinePkg(mgr, tool.GetApkPkg())
	}
	return ensureTool(tool.Binary, tool.GetBrewPkg())
}

func ensureTool(binary, pkg string) error {
	// Check both PATH and the brew opt prefix (uutils tools use libexec/uubin).
	if checkBinary := func() bool {
		if _, err := exec.LookPath(binary); err == nil {
			return true
		}
		prefix := homebrewPrefix()
		if prefix == "" {
			return false
		}
		if _, err := os.Stat(filepath.Join(prefix, "opt", pkg, "libexec", "uubin", binary)); err == nil {
			return true
		}
		return false
	}(); checkBinary {
		return nil
	}

	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("brew not found")
	}

	printInfo("⬇️  Installing %s via Homebrew...", pkg)
	cmd := exec.Command("brew", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	printSuccess("✓ %s installed successfully!", pkg)
	return nil
}

func ensureWindowsTool(binary, pkg string, managers []string) error {
	if _, err := exec.LookPath(binary); err == nil {
		return nil
	}

	candidates := []string{pkg, binary}
	if pkg != strings.ToLower(pkg) {
		candidates = append(candidates, strings.ToLower(pkg))
	}

	seen := map[string]bool{}
	for _, manager := range managers {
		for _, candidate := range candidates {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}

			key := manager + "::" + candidate
			if seen[key] {
				continue
			}
			seen[key] = true

			if err := tryInstallWithWindowsManager(manager, candidate); err == nil {
				printSuccess("✓ %s installed via %s", pkg, manager)
				return nil
			}

			if _, err := exec.LookPath(binary); err == nil {
				printSuccess("✓ %s is already available", pkg)
				return nil
			}
		}
	}

	return fmt.Errorf("no matching package found in available managers")
}

func availableWindowsManagers() []string {
	priority := []string{"winget"}
	available := make([]string, 0, len(priority))
	for _, manager := range priority {
		if _, err := exec.LookPath(manager); err == nil {
			available = append(available, manager)
		}
	}
	return available
}

func tryInstallWithWindowsManager(manager, candidate string) error {
	switch manager {
	case "winget":
		if wingetPackageInstalled(candidate) {
			return nil
		}

		if err := runWingetInstallWithOptionalGsudo("--id", candidate, "--exact", "--source", "winget", "--accept-source-agreements", "--accept-package-agreements", "--silent"); err == nil {
			return nil
		}

		if err := runWingetInstallWithOptionalGsudo("--name", candidate, "--source", "winget", "--accept-source-agreements", "--accept-package-agreements", "--silent"); err == nil {
			return nil
		}

		if wingetPackageInstalled(candidate) {
			return nil
		}

		return fmt.Errorf("winget install failed for %s", candidate)
	default:
		return fmt.Errorf("unsupported manager: %s", manager)
	}
}

func runWingetInstallWithOptionalGsudo(args ...string) error {
	wingetArgs := append([]string{"install"}, args...)

	wingetPath := resolveWindowsExecutable("winget")
	if wingetPath == "" {
		wingetPath = "winget"
	}

	if gsudoPath := resolveWindowsExecutable("gsudo"); gsudoPath != "" {
		commandArgs := append([]string{wingetPath}, wingetArgs...)
		cmd := exec.Command(gsudoPath, commandArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	cmd := exec.Command(wingetPath, wingetArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func primeGsudoCache() error {
	gsudoPath := resolveWindowsExecutable("gsudo")
	if gsudoPath == "" {
		return fmt.Errorf("gsudo not found")
	}

	cmd := exec.Command(gsudoPath, "cache", "on")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveWindowsExecutable(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if localAppData != "" {
		candidate := filepath.Join(localAppData, "Microsoft", "WinGet", "Links", name+".exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return ""
}

func wingetPackageInstalled(candidate string) bool {
	cmd := exec.Command("winget", "list", "--id", candidate, "--exact", "--source", "winget", "--accept-source-agreements")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	text := strings.ToLower(string(out))
	if strings.Contains(text, "no installed package") {
		return false
	}

	return strings.Contains(text, strings.ToLower(candidate))
}

//go:embed resources/psfileicons/PSFileIcons.dll
var psFileIconsDLL []byte

//go:embed resources/psfileicons/PSFileIcons.psm1
var psFileIconsPsm1 string

//go:embed resources/psfileicons/PSFileIcons.psd1
var psFileIconsPsd1 string

//go:embed resources/psfileicons/PSFileIcons.format.ps1xml
var psFileIconsFormatXml string

// ensurePSFileIcons extracts the bundled PSFileIcons module to the user's
// PowerShell module directory. Files are skipped if they are already up-to-date
// (matched by size), so re-running this is cheap.
func ensurePSFileIcons() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	moduleDir := filepath.Join(home, "Documents", "PowerShell", "Modules", "PSFileIcons")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	type fileEntry struct {
		name    string
		content []byte
	}

	files := []fileEntry{
		{"PSFileIcons.dll", psFileIconsDLL},
		{"PSFileIcons.psm1", []byte(psFileIconsPsm1)},
		{"PSFileIcons.psd1", []byte(psFileIconsPsd1)},
		{"PSFileIcons.format.ps1xml", []byte(psFileIconsFormatXml)},
	}

	updated := false
	for _, f := range files {
		dest := filepath.Join(moduleDir, f.name)
		if info, err := os.Stat(dest); err == nil && info.Size() == int64(len(f.content)) {
			continue // already current
		}
		if err := os.WriteFile(dest, f.content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.name, err)
		}
		updated = true
	}

	if updated {
		printSuccess("✓ PSFileIcons module installed")
	}
	return nil
}
