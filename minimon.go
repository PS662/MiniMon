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

type Source struct {
	Path       string `json:"path"`
	SourceType string `json:"source_type"`
}

type Config struct {
	MonitorSources       []Source `json:"monitor_sources"`
	LogDir               string   `json:"log_dir"`
	NotificationInterval int      `json:"notification_interval"`
	LogLevel             string   `json:"log_level"`
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
	config.LogLevel = strings.ToLower(config.LogLevel)

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

func monitorDirectory(path string, interval time.Duration) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create watcher")
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
					log.Info().Int("changes", changeCount).Msg("Accumulating changes in directory")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Err(err).Msg("Watcher error")
			case <-ticker.C:
				if changeCount > 0 {
					notificationMessage := fmt.Sprintf("You have made %d changes in the last %.2f minutes.", changeCount, interval.Minutes())
					log.Info().Msgf(notificationMessage)
					beeep.Notify("MiniMon Notification", notificationMessage, "")
					changeCount = 0
				} else {
					notificationMessage := fmt.Sprintf("Are you saving your work? You have not saved in the last %.2f minutes.", interval.Minutes())
					log.Info().Msgf(notificationMessage)
					beeep.Notify("MiniMon Notification", notificationMessage, "")
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

func monitorGit(filePath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var initialChangeCount int
	var previousChangeCount int
	var totalChangeCount int
	var initialized bool

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
		for range ticker.C {
			currentChangeCount, err := getChangeCount()
			if err != nil {
				continue
			}

			// On the first run, initialize counts
			if !initialized {
				initialChangeCount = currentChangeCount
				previousChangeCount = currentChangeCount
				initialized = true
				log.Info().Msgf("Beginning with %d changes detected by git.", initialChangeCount)
				continue
			}

			// Calculate the difference and update counts
			changeDifference := int(math.Abs(float64(currentChangeCount - previousChangeCount)))
			totalChangeCount += changeDifference
			log.Info().Int("changes", totalChangeCount).Msg("Total changes till now")

			// Log and notify based on the difference
			if changeDifference > 0 {
				log.Info().Int("changes", changeDifference).Msg("Accumulating changes from git monitoring")
				notificationMessage := fmt.Sprintf("You have made %d changes in the last %.2f minutes.", changeDifference, interval.Minutes())
				log.Info().Msgf(notificationMessage)
				beeep.Notify("MiniMon Notification", notificationMessage, "")
			} else {
				notificationMessage := fmt.Sprintf("You have not made any new changes for the last %.2f minutes!!", interval.Minutes())
				log.Info().Msgf(notificationMessage)
				beeep.Notify("MiniMon Notification", notificationMessage, "")
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

	logFile, err := setupLogging(config.LogDir, config.LogLevel)
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
				go monitorDirectory(source.Path, time.Duration(config.NotificationInterval)*time.Second)

			case "git_file", "file":
				if _, err := os.Stat(source.Path); os.IsNotExist(err) {
					log.Warn().Msgf("Invalid source: %s (%s)", source.SourceType, source.Path)
					continue
				}
				if source.SourceType == "git_file" {
					go monitorGit(source.Path, time.Duration(config.NotificationInterval)*time.Second)
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
