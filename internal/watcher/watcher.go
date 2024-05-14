package watcher

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher *fsnotify.Watcher

	Label           string
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
	debug  bool
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	compiledPatterns := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiledPatterns[i] = regexp.MustCompile(pattern)
	}
	return compiledPatterns
}

func NewWatcher(label string, extensions []string, includeDirs []string, ignoreDirs []string, includePatterns []string, ignorePatterns []string, buildCommand string, runCommand *string, rebuildDelay time.Duration, debug bool) (*Watcher, error) {
	// Minimum rebuild delay is 500ms
	if rebuildDelay < 500 {
		rebuildDelay = time.Duration(500)
	}

	// Default values for optional parameters
	if label == "" {
		addedField := false
		separator := ", "
		var sb strings.Builder
		sb.WriteString("[")
		if extensions != nil {
			// if addedField {
			// 	sb.WriteString(separator)
			// }
			fmt.Fprintf(&sb, "E:%v", extensions)
			addedField = true
		}
		if includeDirs != nil {
			if addedField {
				sb.WriteString(separator)
			}
			fmt.Fprintf(&sb, "D:%v", includeDirs)
			addedField = true
		}
		if includePatterns != nil {
			if addedField {
				sb.WriteString(separator)
			}
			fmt.Fprintf(&sb, "P:%v", includePatterns)
			// addedField = true
		}
		sb.WriteString("]")
		label = sb.String()
	}
	if includeDirs == nil {
		includeDirs = []string{"."}
	}
	if ignoreDirs == nil {
		ignoreDirs = []string{}
	}
	if includePatterns == nil {
		includePatterns = []string{}
	}
	if ignorePatterns == nil {
		ignorePatterns = []string{}
	}

	if debug {
		var printRunCmd string
		if runCommand == nil {
			printRunCmd = ""
		} else {
			printRunCmd = *runCommand
		}
		log.Printf(`Creating watcher
	Label: %v

	Include:
	- Extensions  %v
	- Dirs        %v
	- Patterns    %v

	Ignore:
	- Dirs        %v
	- Patterns    %v

	Commands:
	- Build       [%v]
	- Run         [%v]
	`,
			label,
			extensions, includeDirs, includePatterns,
			ignoreDirs, ignorePatterns,
			buildCommand, printRunCmd)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:         fsWatcher,
		Label:           label,
		Extensions:      extensions,
		IncludeDirs:     includeDirs,
		IgnoreDirs:      ignoreDirs,
		IncludePatterns: compilePatterns(includePatterns),
		IgnorePatterns:  compilePatterns(ignorePatterns),
		BuildCommand:    buildCommand,
		RunCommand:      runCommand,
		RebuildDelay:    rebuildDelay,
		debug:           debug,
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
	if w.debug {
		log.Printf("%v - BUILD: %v", w.Label, command)
	}
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (w *Watcher) executeRunCommand(command string) error {
	if w.debug {
		log.Printf("%v - RUN: %v", w.Label, command)
	}
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	w.runCmd = cmd

	err := w.runCmd.Start()
	if err != nil {
		return err
	}

	go func() {
		err := w.runCmd.Wait()
		if err != nil {
			exitError, ok := err.(*exec.ExitError)
			if w.debug && ok && exitError.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL {
				log.Printf("%v - KILLED: %v", w.Label, command)
				return
			} else {
				log.Printf("%v - Run command failed: %v", w.Label, err)
				os.Exit(1)
			}
		}
	}()

	return nil
}

func (w *Watcher) processFileChange() {
	if err := w.executeBuildCommand(w.BuildCommand); err != nil {
		log.Printf("%v - Build command failed: %v", w.Label, err)
	}
	if w.RunCommand != nil {
		if w.runCmd != nil {
			if w.runCmd.ProcessState == nil || !w.runCmd.ProcessState.Exited() {
				if err := w.runCmd.Process.Kill(); err != nil {
					log.Printf("%v - Error killing previous run command: %v", w.Label, err)
					os.Exit(1)
				}
			}
			w.runCmd = nil
		}
		if err := w.executeRunCommand(*w.RunCommand); err != nil {
			log.Printf("%v - Run command failed to start: %v", w.Label, err)
		}
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
			log.Printf("Watcher error: %v", err)
			os.Exit(1)
		}
	}
}

func (w *Watcher) Stop() {
	close(w.stop)
}
