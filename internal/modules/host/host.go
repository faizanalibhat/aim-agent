package host

import (
	"github.com/shirou/gopsutil/v3/host"
)

type HostModule struct{}

type HostData struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Uptime          uint64 `json:"uptime"`
	Procs           uint64 `json:"procs"`
}

func (m *HostModule) Name() string {
	return "host"
}

func (m *HostModule) Gather() (interface{}, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}

	return HostData{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		Uptime:          info.Uptime,
		Procs:           info.Procs,
	}, nil
}
