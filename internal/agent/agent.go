package agent

import (
	"log"
	"os"
	"snapsec-agent/internal/config"
	"snapsec-agent/internal/modules"
	"snapsec-agent/internal/modules/host"
	"snapsec-agent/internal/modules/network"
	"snapsec-agent/pkg/api"
	"time"
)

type Agent struct {
	cfg     *config.Config
	api     *api.Client
	modules []modules.Module
	stop    chan struct{}
}

func NewAgent(cfg *config.Config) *Agent {
	return &Agent{
		cfg: cfg,
		api: api.NewClient(cfg.BackendURL, cfg.APIKey),
		modules: []modules.Module{
			&host.HostModule{},
			&network.NetworkModule{},
		},
		stop: make(chan struct{}),
	}
}

func (a *Agent) RegisterOnly() error {
	assetData, err := a.gatherAll()
	if err != nil {
		return err
	}
	return a.api.Register(assetData)
}

func (a *Agent) Start() error {
	log.Println("Starting Snapsec Agent...")

	// 1. Initial Data Gathering for Registration (Idempotent)
	assetData, err := a.gatherAll()
	if err != nil {
		return err
	}

	// 2. Register Agent (Backend should handle re-registration)
	if err := a.api.Register(assetData); err != nil {
		log.Printf("Warning: Registration during start: %v", err)
	}

	// 3. Start Heartbeat and Asset Reporting Loops
	ticker := time.NewTicker(time.Duration(a.cfg.Interval) * time.Second)
	defer ticker.Stop()

	hostname, _ := os.Hostname()

	for {
		select {
		case <-ticker.C:
			// Send Heartbeat
			if err := a.api.Heartbeat(hostname); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}

			// Gather and Send Assets
			assets, err := a.gatherAll()
			if err != nil {
				log.Printf("Failed to gather assets: %v", err)
				continue
			}

			if err := a.api.SendAssets(assets); err != nil {
				log.Printf("Failed to send assets: %v", err)
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
