package agent

import (
	"fmt"
	"log"
	"os"
	"snapsec-agent/internal/config"
	"snapsec-agent/internal/modules"
	"snapsec-agent/internal/modules/devices"
	"snapsec-agent/internal/modules/hardware"
	"snapsec-agent/internal/modules/host"
	"snapsec-agent/internal/modules/network"
	"snapsec-agent/internal/modules/packages"
	"snapsec-agent/internal/modules/processes"
	"snapsec-agent/internal/modules/security"
	"snapsec-agent/internal/modules/services"
	"snapsec-agent/internal/modules/users"
	"snapsec-agent/pkg/api"
	"time"
)

type Agent struct {
	cfg        *config.Config
	configPath string
	api        *api.Client
	modules    []modules.Module
	stop       chan struct{}
}

func NewAgent(cfg *config.Config, configPath string) *Agent {
	return &Agent{
		cfg:        cfg,
		configPath: configPath,
		api:        api.NewClient(cfg.BackendURL, cfg.APIKey),
		modules: []modules.Module{
			&host.HostModule{},
			&hardware.HardwareModule{},
			&network.NetworkModule{},
			&processes.ProcessesModule{},
			&packages.PackagesModule{},
			&services.ServicesModule{},
			&devices.DevicesModule{},
			&users.UsersModule{},
			&security.SecurityModule{},
		},
		stop: make(chan struct{}),
	}
}

func (a *Agent) RegisterOnly() error {
	hostname, _ := os.Hostname()
	log.Printf("Registering agent with hostname: %s", hostname)

	agentID, err := a.api.Register(hostname)
	if err != nil {
		return err
	}

	log.Printf("Registration successful. Assigned Agent ID: %s", agentID)
	a.cfg.AgentID = agentID

	// Save the agent ID back to the config file
	if err := config.SaveConfig(a.configPath, a.cfg); err != nil {
		return fmt.Errorf("failed to save config with agent ID: %w", err)
	}

	return nil
}

func (a *Agent) Start() error {
	log.Println("Starting Snapsec Agent...")

	// 1. Ensure we have an Agent ID
	if a.cfg.AgentID == "" {
		if err := a.RegisterOnly(); err != nil {
			return fmt.Errorf("failed to register during start: %w", err)
		}
	}

	// 2. Start Heartbeat and Results Reporting Loops
	ticker := time.NewTicker(time.Duration(a.cfg.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send Heartbeat
			if err := a.api.Heartbeat(a.cfg.AgentID); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}

			// Gather and Send Results
			results, err := a.gatherAll()
			if err != nil {
				log.Printf("Failed to gather results: %v", err)
				continue
			}

			if err := a.api.SendResults(a.cfg.AgentID, results); err != nil {
				log.Printf("Failed to send results: %v", err)
			}

		case <-a.stop:
			log.Println("Stopping agent...")
			return nil
		}
	}
}

func (a *Agent) Stop() {
	close(a.stop)
}

func (a *Agent) gatherAll() (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	// Agent info
	payload["agent"] = map[string]string{
		"id":      a.cfg.AgentID,
		"version": config.Version,
	}

	for _, m := range a.modules {
		data, err := m.Gather()
		if err != nil {
			log.Printf("Module %s failed: %v", m.Name(), err)
			continue
		}

		// Special handling for modules that return multiple top-level keys
		if m.Name() == "host_os" {
			if mData, ok := data.(map[string]interface{}); ok {
				for k, v := range mData {
					payload[k] = v
				}
			}
		} else {
			payload[m.Name()] = data
		}
	}

	return payload, nil
}
