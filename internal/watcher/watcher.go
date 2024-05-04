package watcher

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher         *fsnotify.Watcher
	Extensions      []string
	IncludePatterns []*regexp.Regexp
	IgnorePatterns  []*regexp.Regexp
	IncludeDirs     []string
	IgnoreDirs      []string
	BuildCommand    string
	RunCommand      *string
	RebuildDelay    time.Duration
	ChangeDetected  bool

	runCmd *exec.Cmd
	stop   chan struct{}
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	compiledPatterns := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiledPatterns[i] = regexp.MustCompile(pattern)
	}
	return compiledPatterns
}

func NewWatcher(extensions []string, includeDirs []string, ignoreDirs []string, includePatterns []string, ignorePatterns []string, buildCommand string, runCommand *string, rebuildDelay time.Duration) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:         fsWatcher,
		Extensions:      extensions,
		IncludeDirs:     includeDirs,
		IgnoreDirs:      ignoreDirs,
		IncludePatterns: compilePatterns(includePatterns),
		IgnorePatterns:  compilePatterns(ignorePatterns),
		BuildCommand:    buildCommand,
		RunCommand:      runCommand,
		RebuildDelay:    rebuildDelay,
	}

	for _, dir := range w.IncludeDirs {
		if err := w.addDirectoryRecursively(dir); err != nil {
			return nil, err
		}
	}

	w.processFileChange()

	return w, nil
}

func (w *Watcher) addDirectoryRecursively(path string) error {
	// Check if the directory should be ignored
	for _, ignoreDir := range w.IgnoreDirs {
		if ignoreDir == path {
			return nil
		}
	}

	// Check if the path matches any ignore patterns
	for _, pattern := range w.IgnorePatterns {
		if pattern.MatchString(path) {
			return nil
		}
	}

	// Add path, and recursively check subdirectories
	if err := w.watcher.Add(path); err != nil {
		return err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if err := w.addDirectoryRecursively(fullPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Watcher) isValidFile(path string) bool {
	// Ignore directories
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}

	// Check if the path matches any ignore patterns
	for _, pattern := range w.IgnorePatterns {
		if pattern.MatchString(path) {
			return false
		}
	}

	// Check if the file matches any extensions
	ext := filepath.Ext(path)
	for _, e := range w.Extensions {
		if ext == e {
			return true
		}
	}

	// Check if the file matches any include patterns
	for _, pattern := range w.IncludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}

func (w *Watcher) executeBuildCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (w *Watcher) executeRunCommand(command string) (*exec.Cmd, error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to start command: %v\n", err)
		os.Exit(1)
	}
	return cmd, nil
}

func (w *Watcher) processFileChange() {
	if err := w.executeBuildCommand(w.BuildCommand); err != nil {
		log.Printf("Build command failed: %v\n", err)
	}

	if w.RunCommand != nil {
		if w.runCmd != nil {
			_ = w.runCmd.Process.Kill()
		}
		runCmd, err := w.executeRunCommand(*w.RunCommand)
		if err != nil {
			log.Printf("Error executing run command: %v\n", err)
			os.Exit(1)
		}
		w.runCmd = runCmd
	}
}

func debounce(interval time.Duration, fn func()) func() {
	var (
		timer *time.Timer
		mutex sync.Mutex
	)
	return func() {
		mutex.Lock()
		defer mutex.Unlock()

		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(interval, fn)
	}
}

func (w *Watcher) Start() {
	debouncedRebuild := debounce(w.RebuildDelay*time.Millisecond, w.processFileChange)
	defer w.watcher.Close()
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if w.isValidFile(event.Name) {
					debouncedRebuild()
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v\n", err)
			os.Exit(1)
		}
	}
}

func (w *Watcher) Stop() {
	close(w.stop)
}
