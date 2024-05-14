package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/liamrlawrence/observer/internal/watcher"
)

type Config struct {
	InitCommands []string        `json:"init_commands"`
	Watchers     []WatcherConfig `json:"watchers"`
}

type WatcherConfig struct {
	Label           string        `json:"label,omitempty"`
	Extensions      []string      `json:"extensions"`
	IncludeDirs     []string      `json:"include_dirs,omitempty"`
	IgnoreDirs      []string      `json:"ignore_dirs,omitempty"`
	IncludePatterns []string      `json:"include_patterns,omitempty"`
	IgnorePatterns  []string      `json:"ignore_patterns,omitempty"`
	BuildCommand    string        `json:"build_command"`
	RunCommand      *string       `json:"run_command,omitempty"`
	RebuildDelay    time.Duration `json:"rebuild_delay,omitempty"`
	Debug           bool          `json:"debug"`
}

func main() {
	configFile := flag.String("config", "observer.config.json", "path to config file")
	debugMode := flag.Bool("debug", false, "Force debug status for all watchers")
	flag.Parse()

	config, err := readConfig(*configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v", err)
		return
	}

	printSplash()

	for _, cmdStr := range config.InitCommands {
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Printf("INIT: %v", cmdStr)
		if err := cmd.Run(); err != nil {
			log.Printf("Error executing init command '%v': %v", cmdStr, err)
			os.Exit(1)
		}
	}

	for _, watcherConfig := range config.Watchers {
		if *debugMode {
			watcherConfig.Debug = *debugMode
		}

		w, err := watcher.NewWatcher(
			watcherConfig.Label,
			watcherConfig.Extensions,
			watcherConfig.IncludeDirs,
			watcherConfig.IgnoreDirs,
			watcherConfig.IncludePatterns,
			watcherConfig.IgnorePatterns,
			watcherConfig.BuildCommand,
			watcherConfig.RunCommand,
			watcherConfig.RebuildDelay,
			watcherConfig.Debug)
		if err != nil {
			log.Printf("Error initializing watcher: %v", err)
			os.Exit(1)
		}
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

func printSplash() {
	var splash = `
    .-.     ___    oo_     wWw ()_()wWw    wWw wWw ()_()
  c(O_O)c  (___)__/  _)-<  (O)_(O o)(O)    (O) (O)_(O o)
 ,'.---.` + "`" + `, (O)(O) \__ ` + "`" + `.   / __)|^_\( \    / ) / __)|^_\
/ /|_|_|\ \/  _\     ` + "`" + `. | / (   |(_))\ \  / / / (   |(_))
| \_____/ || |_))    _| |(  _)  |  / /  \/  \(  _)  |  /
'. ` + "`" + `---' .` + "`" + `| |_)) ,-'   | \ \_  )|\\ \ ` + "`" + `--' / \ \_  )|\\
  ` + "`" + `-...-'  (.'-' (_..--'   \__)(/  \) ` + "`" + `-..-'   \__)(/  \)

`
	fmt.Printf(splash)
}
