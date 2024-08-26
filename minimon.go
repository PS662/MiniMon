package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
)

type Config struct {
	MonitorSources       map[string]string `json:"monitor_sources"`
	LogDir               string            `json:"log_dir"`
	NotificationInterval int               `json:"notification_interval"`
}

func loadConfig(configPath string) (*Config, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func monitorDirectory(path string, interval time.Duration) {
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
					remainingTime := time.Until(time.Now().Add(interval)).Minutes()
					log.Printf("Accumulating change count: %d. Next notification in %.2f minutes.\n", changeCount, remainingTime)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
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

	for srcName, srcPath := range config.MonitorSources {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			log.Printf("Invalid source: %s (%s)", srcName, srcPath)
			continue
		}

		go monitorDirectory(srcPath, time.Duration(config.NotificationInterval)*time.Second)
	}

	// Keep the main function running
	select {}
}
