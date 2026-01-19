package packages

import (
	"os/exec"
	"runtime"
	"strings"
)

type PackagesModule struct{}

type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (m *PackagesModule) Name() string {
	return "packages"
}

func (m *PackagesModule) Gather() (interface{}, error) {
	switch runtime.GOOS {
	case "linux":
		return m.gatherLinux()
	case "darwin":
		return m.gatherDarwin()
	case "windows":
		return m.gatherWindows()
	default:
		return nil, nil
	}
}

func (m *PackagesModule) gatherLinux() ([]PackageInfo, error) {
	// Try dpkg (Debian/Ubuntu)
	if _, err := exec.LookPath("dpkg"); err == nil {
		out, _ := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version}\n").Output()
		return parseLines(string(out)), nil
	}
	// Try rpm (CentOS/RHEL)
	if _, err := exec.LookPath("rpm"); err == nil {
		out, _ := exec.Command("rpm", "-qa", "--queryformat", "%{NAME} %{VERSION}\n").Output()
		return parseLines(string(out)), nil
	}
	return nil, nil
}

func (m *PackagesModule) gatherDarwin() ([]PackageInfo, error) {
	// Try brew
	if _, err := exec.LookPath("brew"); err == nil {
		out, _ := exec.Command("brew", "list", "--versions").Output()
		return parseLines(string(out)), nil
	}
	return nil, nil
}

func (m *PackagesModule) gatherWindows() ([]PackageInfo, error) {
	// Use powershell to get installed software
	cmd := "Get-ItemProperty HKLM:\\Software\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\* | Select-Object DisplayName, DisplayVersion | ForEach-Object { \"$($_.DisplayName) $($_.DisplayVersion)\" }"
	out, _ := exec.Command("powershell", "-Command", cmd).Output()
	return parseLines(string(out)), nil
}

func parseLines(output string) []PackageInfo {
	var pkgs []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pkgs = append(pkgs, PackageInfo{
				Name:    parts[0],
				Version: parts[1],
			})
		} else if len(parts) == 1 {
			pkgs = append(pkgs, PackageInfo{
				Name:    parts[0],
				Version: "unknown",
			})
		}
	}
	return pkgs
}
