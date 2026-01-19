package service

import (
	"log"
	"fmt"
	"snapsec-agent/internal/agent"
	"snapsec-agent/internal/config"

	"github.com/kardianos/service"
)

type program struct {
	agent *agent.Agent
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work in a separate goroutine.
	go p.run()
	return nil
}

func (p *program) run() {
	if err := p.agent.Start(); err != nil {
		log.Printf("Agent error: %v", err)
	}
}

func (p *program) Stop(s service.Service) error {
	p.agent.Stop()
	return nil
}

func newService(cfg *config.Config, configPath string) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "SnapsecAgent",
		DisplayName: "Snapsec Security Agent",
		Description: "Monitors system assets and sends data to Snapsec backend.",
	}

	a := agent.NewAgent(cfg, configPath)
	prg := &program{agent: a}
	return service.New(prg, svcConfig)
}

func AutoHandle(cfg *config.Config, configPath string) error {
	s, err := newService(cfg, configPath)
	if err != nil {
		return err
	}

	// Check if we are running interactively or as a service
	if service.Interactive() {
		status, err := s.Status()
		if err != nil || status == service.StatusUnknown {
			log.Println("Agent not installed. Starting installation process...")

			// 1. Register with backend
			a := agent.NewAgent(cfg, configPath)
			log.Println("Registering agent with backend...")
			if err := a.RegisterOnly(); err != nil {
				return fmt.Errorf("registration failed (installation aborted): %w", err)
			}
			log.Println("Registration successful.")

			// 2. Install as service
			log.Println("Installing service...")
			if err := s.Install(); err != nil {
				return fmt.Errorf("failed to install service: %w", err)
			}

			// 3. Start service
			log.Println("Starting service...")
			if err := s.Start(); err != nil {
				return fmt.Errorf("failed to start service: %w", err)
			}

			log.Println("Snapsec Agent installed and started successfully.")
			return nil
		}

		// If already installed but run interactively, maybe the user wants to manage it?
		// For now, we'll just run it interactively to allow debugging.
		log.Println("Agent already installed. Running in interactive mode...")
	}

	return s.Run()
}
