package vulnscan

import (
	"context"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

type ScanManager struct {
	plugins       map[string]ScannerPlugin
	config        PluginConfig
	scanInterval  time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	resultHandler func([]NormalizedFinding)
}

func NewScanManager(config PluginConfig, handler func([]NormalizedFinding)) *ScanManager {
	return &ScanManager{
		plugins:       make(map[string]ScannerPlugin),
		config:        config,
		stopCh:        make(chan struct{}),
		resultHandler: handler,
	}
}

func (m *ScanManager) RegisterPlugin(name string, plugin ScannerPlugin) error {
	if err := plugin.Init(m.config); err != nil {
		return err
	}
	m.plugins[name] = plugin
	return nil
}

func (m *ScanManager) SetScanInterval(intervalSeconds int) {
	if intervalSeconds > 0 {
		m.scanInterval = time.Duration(intervalSeconds) * time.Second
	} else {
		m.scanInterval = 0
	}
}

func (m *ScanManager) Start() {
	m.wg.Add(1)
	go m.runLoop()
}

func (m *ScanManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	for _, plugin := range m.plugins {
		plugin.Cleanup()
	}
}

func (m *ScanManager) runLoop() {
	defer m.wg.Done()
	
	if m.scanInterval <= 0 {
		return
	}
	
	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.RunScheduledScan()
		case <-m.stopCh:
			return
		}
	}
}

func getLocalScanTargets() []string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			"C:\\Windows\\System32\\drivers\\etc",
			"C:\\ProgramData",
			"C:\\inetpub",
			"C:\\Users",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
		}
	case "darwin":
		candidates = []string{
			"/etc",
			"/Library",
			"/Applications",
			"/Users",
			"/usr/local/etc",
		}
	default:
		candidates = []string{
			"/etc",
			"/usr/local/etc",
			"/opt",
			"/var/www",
			"/home",
			"/var/lib/docker",
			"/etc/kubernetes",
		}
	}

	var targets []string
	for _, dir := range candidates {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			targets = append(targets, dir)
		}
	}
	return targets
}

func (m *ScanManager) RunScheduledScan() {
	log.Println("Starting scheduled vulnerability scan...")
	
	targets := getLocalScanTargets()
	if len(targets) == 0 {
		targets = []string{"localhost"}
	}

	for name, plugin := range m.plugins {
		job := ScanJob{
			ID:      "scheduled-" + name + "-" + time.Now().Format("20060102150405"),
			Tool:    name,
			Targets: targets,
			Options: map[string]string{
				"protocol": "file",
				"tags":     "secrets,keys,tokens,credentials,misconfiguration",
			},
		}
		
		go m.RunJob(plugin, job)
	}
}

func (m *ScanManager) RunJobs(jobs []ScanJob) {
	for _, job := range jobs {
		plugin, ok := m.plugins[job.Tool]
		if !ok {
			log.Printf("Cannot run job %s: Tool %s not registered", job.ID, job.Tool)
			continue
		}
		go m.RunJob(plugin, job)
	}
}

func (m *ScanManager) RunJob(plugin ScannerPlugin, job ScanJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	log.Printf("Executing scan job %s with tool %s", job.ID, job.Tool)
	result, err := plugin.Execute(ctx, job)
	if err != nil {
		log.Printf("Scan job %s failed: %v", job.ID, err)
		return
	}

	log.Printf("Scan job %s completed with %d findings", job.ID, len(result.Findings))
	if m.resultHandler != nil {
		m.resultHandler(result.Findings)
	}
}

func (m *ScanManager) RunCLI(tool string, target string) {
	var targets []string
	if target == "" {
		targets = getLocalScanTargets()
		if len(targets) == 0 {
			if runtime.GOOS == "windows" {
				targets = []string{"C:\\"}
			} else {
				targets = []string{"."}
			}
		}
	} else {
		targets = []string{target}
	}

	for name, plugin := range m.plugins {
		if tool != "" && name != tool {
			continue
		}

		job := ScanJob{
			ID:      "cli-" + name + "-" + time.Now().Format("20060102150405"),
			Tool:    name,
			Targets: targets,
			Options: map[string]string{
				"protocol": "file",
				"tags":     "secrets,keys,tokens,credentials,misconfiguration",
				"verbose":  "true",
			},
		}

		// Run synchronously for CLI
		m.wg.Add(1)
		func() {
			defer m.wg.Done()
			m.RunJob(plugin, job)
		}()
	}
}
