package packages

import (
	"os/exec"
	"runtime"
	"strings"
)

type PackagesModule struct{}

type PackageInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Arch        string `json:"arch"`
	InstalledAt string `json:"installed_at"`
}

type PackagesData struct {
	Type  string        `json:"type"`
	Count int           `json:"count"`
	List  []PackageInfo `json:"list"`
}

func (m *PackagesModule) Name() string {
	return "packages"
}

func (m *PackagesModule) Gather() (interface{}, error) {
	var pkgType string
	var list []PackageInfo

	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("dpkg"); err == nil {
			pkgType = "apt"
			out, _ := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version} ${Architecture}\n").Output()
			list = parseLines(string(out))
		} else if _, err := exec.LookPath("rpm"); err == nil {
			pkgType = "rpm"
			out, _ := exec.Command("rpm", "-qa", "--queryformat", "%{NAME} %{VERSION} %{ARCH}\n").Output()
			list = parseLines(string(out))
		}
	case "darwin":
		pkgType = "brew"
		out, _ := exec.Command("brew", "list", "--versions").Output()
		list = parseLines(string(out))
	case "windows":
		pkgType = "windows"
		// Simplified
	}

	return PackagesData{
		Type:  pkgType,
		Count: len(list),
		List:  list,
	}, nil
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
		if len(parts) >= 3 {
			pkgs = append(pkgs, PackageInfo{
				Name:    parts[0],
				Version: parts[1],
				Arch:    parts[2],
			})
		} else if len(parts) >= 2 {
			pkgs = append(pkgs, PackageInfo{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}
	return pkgs
}
