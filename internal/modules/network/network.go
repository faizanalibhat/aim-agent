package network

import (
	"github.com/shirou/gopsutil/v3/net"
)

type NetworkModule struct{}

type InterfaceData struct {
	Name         string   `json:"name"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_addr"`
	Flags        []string `json:"flags"`
	Addrs        []string `json:"addrs"`
}

func (m *NetworkModule) Name() string {
	return "network"
}

func (m *NetworkModule) Gather() (interface{}, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var data []InterfaceData
	for _, i := range interfaces {
		var addrs []string
		for _, a := range i.Addrs {
			addrs = append(addrs, a.Addr)
		}

		data = append(data, InterfaceData{
			Name:         i.Name,
			MTU:          i.MTU,
			HardwareAddr: i.HardwareAddr,
			Flags:        i.Flags,
			Addrs:        addrs,
		})
	}

	return data, nil
}
