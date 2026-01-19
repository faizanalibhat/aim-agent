package agent

import (
	"fmt"
	"log"
	"os"
	"snapsec-agent/internal/config"
	"snapsec-agent/internal/modules"
	"snapsec-agent/internal/modules/host"
	"snapsec-agent/internal/modules/network"
	"snapsec-agent/internal/modules/packages"
	"snapsec-agent/internal/modules/processes"
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
			&network.NetworkModule{},
			&packages.PackagesModule{},
			&processes.ProcessesModule{},
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
	results := make(map[string]interface{})
	for _, m := range a.modules {
		data, err := m.Gather()
		if err != nil {
			log.Printf("Module %s failed: %v", m.Name(), err)
			continue
		}
		results[m.Name()] = data
	}

	return results, nil
}
