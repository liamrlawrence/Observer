package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/liamrlawrence/observer/internal/watcher"
)

type Config struct {
	RefreshRate time.Duration   `json:"refresh_rate,omitempty"`
	Watchers    []WatcherConfig `json:"watchers"`
}

type WatcherConfig struct {
	Extensions      []string `json:"extensions"`
	IncludeDirs     []string `json:"include_dirs,omitempty"`
	IgnoreDirs      []string `json:"ignore_dirs,omitempty"`
	IncludePatterns []string `json:"include_patterns,omitempty"`
	IgnorePatterns  []string `json:"ignore_patterns,omitempty"`
	BuildCommand    string   `json:"build_command"`
	RunCommand      *string  `json:"run_command,omitempty"`
}

func main() {
	configFile := flag.String("config", "observer.config.json", "path to config file")
	flag.Parse()

	config, err := readConfig(*configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return
	}

	// Default values for optional top-level configuration parameters
	if config.RefreshRate == 0 {
		config.RefreshRate = time.Duration(500)
	}

	for _, watcherConfig := range config.Watchers {
		// Default values for optional watcher configuration parameters
		if watcherConfig.IncludeDirs == nil {
			watcherConfig.IncludeDirs = []string{"."}
		}
		if watcherConfig.IgnoreDirs == nil {
			watcherConfig.IgnoreDirs = []string{}
		}
		if watcherConfig.IncludePatterns == nil {
			watcherConfig.IncludePatterns = []string{}
		}
		if watcherConfig.IgnorePatterns == nil {
			watcherConfig.IgnorePatterns = []string{}
		}

		w := watcher.NewWatcher(
			watcherConfig.Extensions,
			watcherConfig.IncludeDirs,
			watcherConfig.IgnoreDirs,
			watcherConfig.IncludePatterns,
			watcherConfig.IgnorePatterns,
			watcherConfig.BuildCommand,
			watcherConfig.RunCommand,
			config.RefreshRate)
		go w.Start()
	}

	select {}
}

func readConfig(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
