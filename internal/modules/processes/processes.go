package processes

import (
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessesModule struct{}

type ProcessInfo struct {
	Pid      int32   `json:"pid"`
	Name     string  `json:"name"`
	Username string  `json:"username"`
	CPU      float64 `json:"cpu"`
	Memory   float32 `json:"memory"`
	Status   string  `json:"status"`
}

func (m *ProcessesModule) Name() string {
	return "processes"
}

func (m *ProcessesModule) Gather() (interface{}, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var results []ProcessInfo
	for _, p := range procs {
		name, _ := p.Name()
		username, _ := p.Username()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		statusStr := "unknown"
		if len(status) > 0 {
			statusStr = status[0]
		}

		results = append(results, ProcessInfo{
			Pid:      p.Pid,
			Name:     name,
			Username: username,
			CPU:      cpu,
			Memory:   mem,
			Status:   statusStr,
		})
	}

	return results, nil
}
