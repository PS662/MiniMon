package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
)

type Config struct {
	MonitorSources       []string `json:"monitor_sources"`
	LogDir               string   `json:"log_dir"`
	NotificationInterval int      `json:"notification_interval"`
	LogLevel             string   `json:"log_level"` // New field for log level
}

// Log level constants
const (
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
)

func loadConfig(configPath string) (*Config, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	// Normalize log level to lowercase
	config.LogLevel = strings.ToLower(config.LogLevel)

	// Default log level to "warning" if not set or invalid
	if config.LogLevel != LevelInfo && config.LogLevel != LevelWarning && config.LogLevel != LevelError {
		config.LogLevel = LevelWarning
	}

	return &config, nil
}

func setupLogging(logDir string) (*os.File, error) {
	if logDir == "" {
		return nil, nil
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("log directory does not exist: %s", logDir)
	}

	logFilePath := filepath.Join(logDir, "minimon.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %v", err)
	}

	log.SetOutput(logFile)
	return logFile, nil
}

func shouldLog(logLevel, configLevel string) bool {
	// Order: error < warning < info
	logLevels := map[string]int{
		LevelError:   0,
		LevelWarning: 1,
		LevelInfo:    2,
	}

	return logLevels[logLevel] <= logLevels[configLevel]
}

func monitorDirectory(path string, interval time.Duration, logDir, logLevel string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	changeCount := 0
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					changeCount++
					if logDir != "" && shouldLog(LevelInfo, logLevel) {
						remainingTime := interval.Minutes()
						log.Printf("Accumulating change count: %d. Next notification in %.2f minutes.\n", changeCount, remainingTime)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if shouldLog(LevelError, logLevel) {
					log.Println("Error:", err)
				}
			case <-ticker.C:
				// Notify after every interval
				if changeCount > 0 {
					notificationMessage := fmt.Sprintf("You have made %d saves in the last %d seconds. Remember to save your work", changeCount, int(interval.Seconds()))
					beeep.Notify("MiniMon Notification", notificationMessage, "")
					changeCount = 0 // Reset count after notification
				}
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	// Keep the goroutine running
	select {}
}

func main() {
	configPath := os.Getenv("MINIMON_CONFIG")
	if configPath == "" {
		configPath = "/usr/minimon/config.json"
	}

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Setup logging
	logFile, err := setupLogging(config.LogDir)
	if err != nil {
		log.Printf("Warning: %v. Skipping file logging.\n", err)
	} else if logFile != nil {
		defer logFile.Close()
	}

	for _, srcPath := range config.MonitorSources {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			if shouldLog(LevelWarning, config.LogLevel) {
				log.Printf("Invalid source: %s", srcPath)
			}
			continue
		}

		go monitorDirectory(srcPath, time.Duration(config.NotificationInterval)*
			time.Second, config.LogDir, config.LogLevel)
	}

	// Keep the main function running
	select {}
}
