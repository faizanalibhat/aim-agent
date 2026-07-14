package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"snapsec-agent/internal/config"
	"snapsec-agent/internal/service"
	"snapsec-agent/internal/vulnscan"
	"snapsec-agent/internal/vulnscan/nuclei"
	"snapsec-agent/pkg/api"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "scan" {
		scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
		toolFlag := scanCmd.String("tool", "", "Specific tool to run (e.g., nuclei)")
		outputFlag := scanCmd.String("output", "", "Output file for JSON results")
		targetFlag := scanCmd.String("target", "", "Target to scan (e.g. /etc, C:\\, or example.com)")
		configPath := scanCmd.String("config", config.GetDefaultConfigPath(), "Path to configuration file")

		scanCmd.Parse(os.Args[2:])

		cfg, err := config.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		apiClient := api.NewClient(cfg.BackendURL, cfg.APIKey)

		pluginCfg := vulnscan.PluginConfig{
			BinDir:      "/var/lib/snapsec/bin",
			TemplateDir: "/var/lib/snapsec/templates",
		}
		if runtime.GOOS == "windows" {
			pluginCfg.BinDir = "C:\\ProgramData\\snapsec-agent\\bin"
			pluginCfg.TemplateDir = "C:\\ProgramData\\snapsec-agent\\templates"
		}

		manager := vulnscan.NewScanManager(pluginCfg, func(findings []vulnscan.NormalizedFinding) {
			if *outputFlag != "" {
				b, _ := json.MarshalIndent(findings, "", "  ")
				if err := os.WriteFile(*outputFlag, b, 0644); err != nil {
					log.Printf("Failed to write output file: %v", err)
				} else {
					log.Printf("Results written to %s", *outputFlag)
				}
			} else {
				if cfg.AgentID == "" {
					log.Println("AgentID not set, cannot send to AIM. Please register the agent first.")
					return
				}
				if _, err := apiClient.SendVulnerabilities(cfg.AgentID, findings); err != nil {
					log.Printf("Failed to send vulnerabilities to AIM: %v", err)
				} else {
					log.Println("Results successfully sent to AIM.")
				}
			}
		})

		if err := manager.RegisterPlugin("nuclei", &nuclei.NucleiScanner{}); err != nil {
			log.Fatalf("Failed to initialize nuclei plugin: %v", err)
		}

		manager.RunCLI(*toolFlag, *targetFlag)
		manager.Stop()
		return
	}

	configPath := flag.String("config", config.GetDefaultConfigPath(), "Path to configuration file")
	foreground := flag.Bool("f", false, "Run in foreground (interactive mode)")
	flag.BoolVar(foreground, "foreground", false, "Run in foreground (interactive mode)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)

	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Printf("Please ensure config exists at %s or specify path with -config\n", *configPath)
		os.Exit(1)
	}

	// AutoHandle will install/register if not installed, or run if already installed.
	if err := service.AutoHandle(cfg, *configPath, *foreground); err != nil {
		log.Fatal(err)
	}
}
