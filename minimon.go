package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Notification struct {
	NotificationHead string `json:"notification_head"`
	OnChange         string `json:"on_change"`
	OnIdle           string `json:"on_idle"`
	NotificationTail string `json:"notification_tail"`
	IsIdle           bool   `json:"is_idle"`
	IsIdleText       string `json:"is_idle_text"`
	IsChange         bool   `json:"is_change"`
	IsChangeText     string `json:"is_change_text"`
}

type NotificationConfig struct {
	NotificationInterval int            `json:"notification_interval"`
	NotificationSet      []Notification `json:"notification_set"`
	MaxIdleTime          int            `json:"max_idle_time"`
}

type Source struct {
	Path               string             `json:"path"`
	SourceType         string             `json:"source_type"`
	NotificationConfig NotificationConfig `json:"notification_config"`
}

type MonitorProps struct {
	LogDir   string `json:"log_dir"`
	LogLevel string `json:"log_level"`
}

type Config struct {
	MonitorSources []Source     `json:"monitor_sources"`
	MonitorProps   MonitorProps `json:"monitor_props"`
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

	// Normalize log level to lowercase
	config.MonitorProps.LogLevel = strings.ToLower(config.MonitorProps.LogLevel)

	// Set notification flags based on the configuration
	for i := range config.MonitorSources {
		for j := range config.MonitorSources[i].NotificationConfig.NotificationSet {
			notification := &config.MonitorSources[i].NotificationConfig.NotificationSet[j]
			notification.IsChange = false
			notification.IsIdle = false
			if notification.OnChange != "" {
				notification.IsChange = true
				notification.IsChangeText = notification.OnChange
			}
			if notification.OnIdle != "" {
				notification.IsIdle = true
				notification.IsIdleText = notification.OnIdle
			}
		}
	}

	return &config, nil
}

func setupLogging(logDir, logLevel string) (*os.File, error) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	var logFile *os.File
	var err error

	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "console":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	default:
		if logDir != "" {
			if _, err := os.Stat(logDir); os.IsNotExist(err) {
				return nil, fmt.Errorf("log directory does not exist: %s", logDir)
			}

			logFilePath := filepath.Join(logDir, "minimon.log")
			logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("could not open log file: %v", err)
			}

			log.Logger = log.Output(logFile)
		}
	}

	return logFile, err
}

func constructNotificationMessage(notification Notification, changeCount int, timeInterval float64, onChange bool) string {
	if onChange && notification.IsChangeText != "" {
		return fmt.Sprintf("%s %d %s %.2f minutes. %s",
			notification.NotificationHead, changeCount, notification.IsChangeText, timeInterval, notification.NotificationTail)
	} else if !onChange && notification.IsIdleText != "" {
		return fmt.Sprintf("%s %s %.2f minutes %s",
			notification.NotificationHead, notification.IsIdleText, timeInterval, notification.NotificationTail)
	}
	// Default notification message if all fields are empty or absent
	if onChange {
		return fmt.Sprintf("activity notification: %d changes in %.2f minutes", changeCount, timeInterval)
	}
	return fmt.Sprintf("idle notification: idle time: %.2f minutes", timeInterval)
}

func monitorDirectory(path string, config NotificationConfig) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create watcher")
	}
	defer watcher.Close()

	changeCount := 0
	idleTime := 0.0
	intervalTime := float64(config.NotificationInterval) / 60.0
	ticker := time.NewTicker(time.Duration(config.NotificationInterval) * time.Second)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					changeCount++
					log.Info().Int("changes", changeCount).Msg("Accumulating changes in directory")
					idleTime = 0 // Reset idle time when a change is detected
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Err(err).Msg("Watcher error")
			case <-ticker.C:
				if changeCount > 0 {
					//log.Info().Msgf("Change detected, preparing to send change notifications. Change count: %d", changeCount)
					for _, notification := range config.NotificationSet {
						//log.Info().Msgf("Processing notification %d: %+v", i+1, notification)
						if notification.IsChange {
							notificationMessage := constructNotificationMessage(notification, changeCount, intervalTime, true)
							//log.Info().Msgf("Sending change notification: %s", notificationMessage)
							err := beeep.Notify("MiniMon Notification", notificationMessage, "")
							if err != nil {
								log.Error().Err(err).Msg("Failed to send change notification")
							}
						}
					}
					changeCount = 0
				} else {
					idleTime += intervalTime
					log.Info().Msgf("No changes detected, idle time: %.2f minutes", idleTime)
					if idleTime >= float64(config.MaxIdleTime)/60 {
						log.Info().Msg("Max idle time reached, stopping notifications.")
						continue
					}
					for _, notification := range config.NotificationSet {
						//log.Info().Msgf("Processing notification %d: %+v", i+1, notification)
						if notification.IsIdle {
							notificationMessage := constructNotificationMessage(notification, changeCount, idleTime, false)
							//log.Info().Msgf("Sending idle notification: %s", notificationMessage)
							err := beeep.Notify("MiniMon Notification", notificationMessage, "")
							if err != nil {
								log.Error().Err(err).Msg("Failed to send idle notification")
							}
						}
					}
				}

			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to add directory to watcher")
	}

	select {}
}

func monitorGit(filePath string, config NotificationConfig) {
	ticker := time.NewTicker(time.Duration(config.NotificationInterval) * time.Second)
	defer ticker.Stop()

	var initialChangeCount int
	var previousChangeCount int
	var totalChangeCount int
	idleTime := 0.0
	intervalTime := float64(config.NotificationInterval) / 60.0

	// Function to fetch the current change count using git diff
	getChangeCount := func() (int, error) {
		cmdGetRepoPath := exec.Command("git", "rev-parse", "--show-toplevel")
		cmdGetRepoPath.Dir = filepath.Dir(filePath)
		var repoPathOut bytes.Buffer
		cmdGetRepoPath.Stdout = &repoPathOut
		err := cmdGetRepoPath.Run()
		if err != nil {
			log.Error().Err(err).Msg("Failed to determine Git repository path")
			return 0, err
		}

		gitRepoPath := strings.TrimSpace(repoPathOut.String())

		if err := os.Chdir(gitRepoPath); err != nil {
			log.Error().Err(err).Msgf("Failed to change directory to %s", gitRepoPath)
			return 0, err
		}

		// Run git diff to check for changes
		cmd := exec.Command("git", "diff", "--numstat", "HEAD", filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()

		// Handle exit status 1 (no differences found)
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
				log.Info().Msg("No changes detected by git diff")
				return 0, nil
			} else {
				log.Error().Err(err).Msg("Failed to run git diff")
				return 0, err
			}
		}

		// Parse the output to count the number of lines changed
		lines := strings.Split(out.String(), "\n")
		changeCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				added, _ := strconv.Atoi(fields[0])
				removed, _ := strconv.Atoi(fields[1])
				changeCount += added + removed
			}
		}
		return changeCount, nil
	}

	go func() {
		// Perform the initial check immediately
		currentChangeCount, err := getChangeCount()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get initial change count")
			return
		}

		// Initialize counts
		initialChangeCount = currentChangeCount
		previousChangeCount = currentChangeCount
		log.Info().Msgf("Beginning with %d changes detected by git.", initialChangeCount)

		for range ticker.C {
			currentChangeCount, err := getChangeCount()
			if err != nil {
				continue
			}

			// Calculate the difference and update counts
			changeDifference := int(math.Abs(float64(currentChangeCount - previousChangeCount)))
			totalChangeCount += changeDifference
			log.Info().Int("changes", totalChangeCount).Msg("Total changes till now")

			if changeDifference > 0 {
				for _, notification := range config.NotificationSet {
					if notification.IsChange {
						notificationMessage := constructNotificationMessage(notification, changeDifference, intervalTime, true)
						log.Info().Msgf(notificationMessage)
						beeep.Notify("MiniMon Notification", notificationMessage, "")
					}
				}
				idleTime = 0 // Reset idle time when changes are detected
			} else {
				idleTime += intervalTime
				log.Info().Msgf("No changes detected, idle time: %.2f minutes", idleTime)
				if idleTime >= float64(config.MaxIdleTime)/60 {
					log.Info().Msg("Max idle time reached, suppressing further idle notifications.")
					continue
				}
				for _, notification := range config.NotificationSet {
					if notification.IsIdle {
						notificationMessage := constructNotificationMessage(notification, changeDifference, idleTime, false)
						log.Info().Msgf(notificationMessage)
						beeep.Notify("MiniMon Notification", notificationMessage, "")
					}
				}
			}

			// Update the previousChangeCount
			previousChangeCount = currentChangeCount
		}
	}()

	select {}
}

func main() {
	configPath := os.Getenv("MINIMON_CONFIG")
	if configPath == "" {
		configPath = "/usr/minimon/config.json"
	}

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	logFile, err := setupLogging(config.MonitorProps.LogDir, config.MonitorProps.LogLevel)
	if err != nil {
		log.Warn().Msgf("Warning: %v. Skipping file logging.", err)
	} else if logFile != nil {
		defer logFile.Close()
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	doneChan := make(chan struct{})

	go func() {
		for _, source := range config.MonitorSources {
			switch source.SourceType {
			case "dir":
				if _, err := os.Stat(source.Path); os.IsNotExist(err) {
					log.Warn().Msgf("Invalid source: %s (%s)", source.SourceType, source.Path)
					continue
				}
				go monitorDirectory(source.Path, source.NotificationConfig)

			case "git_file", "file":
				if _, err := os.Stat(source.Path); os.IsNotExist(err) {
					log.Warn().Msgf("Invalid source: %s (%s)", source.SourceType, source.Path)
					continue
				}
				if source.SourceType == "git_file" {
					go monitorGit(source.Path, source.NotificationConfig)
				}

			default:
				log.Warn().Msgf("Unsupported source type: %s", source.SourceType)
			}
		}

		// Blocking wait until the stop signal is received
		<-stopChan
		log.Info().Msg("Shutting down MiniMon...")

		// Perform cleanup and exit
		close(doneChan)
	}()

	// Wait until graceful shutdown is completed
	<-doneChan
	log.Info().Msg("MiniMon exited gracefully.")
}
