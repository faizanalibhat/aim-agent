package vulnscan

import (
	"context"
	"log"
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

func (m *ScanManager) RunScheduledScan() {
	log.Println("Starting scheduled vulnerability scan...")
	
	for name, plugin := range m.plugins {
		// Create a generic job for the local host. For real use, targets might be based on local interfaces/services.
		job := ScanJob{
			ID:      "scheduled-" + name + "-" + time.Now().Format("20060102150405"),
			Tool:    name,
			Targets: []string{"localhost"},
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
	if len(result.Findings) > 0 && m.resultHandler != nil {
		m.resultHandler(result.Findings)
	}
}
