package watcher

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

type FileData struct {
	Path    string
	ModTime time.Time
}

type Watcher struct {
	Extensions      []string
	IncludeDirs     []string
	IgnoreDirs      []string
	IncludePatterns []*regexp.Regexp
	IgnorePatterns  []*regexp.Regexp
	BuildCommand    string
	RunCommand      *string
	RefreshRate     time.Duration
	FileData        map[string]*FileData
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

func NewWatcher(extensions []string, includeDirs []string, ignoreDirs []string, includePatterns []string, ignorePatterns []string, buildCommand string, runCommand *string, refreshRate time.Duration) *Watcher {
	compiledIncludePatterns := compilePatterns(includePatterns)
	compiledIgnorePatterns := compilePatterns(ignorePatterns)
	return &Watcher{
		Extensions:      extensions,
		IncludeDirs:     includeDirs,
		IgnoreDirs:      ignoreDirs,
		IncludePatterns: compiledIncludePatterns,
		IgnorePatterns:  compiledIgnorePatterns,
		BuildCommand:    buildCommand,
		RunCommand:      runCommand,
		RefreshRate:     refreshRate,
		FileData:        make(map[string]*FileData),
		ChangeDetected:  false,
		stop:            make(chan struct{}),
	}
}

func (w *Watcher) visitFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Printf("Error accessing path %s: %v\n", path, err)
		os.Exit(1)
	}

	// Check if the directory should be ignored
	if info.IsDir() {
		dirName := filepath.Base(path)
		for _, ignoreDir := range w.IgnoreDirs {
			if dirName == ignoreDir {
				return filepath.SkipDir
			}
		}
		return nil
	}

	// Check if the path matches any ignore patterns
	for _, pattern := range w.IgnorePatterns {
		if pattern.MatchString(path) {
			return nil
		}
	}

	// Check if the file matches any extensions
	matchesExtension := false
	ext := filepath.Ext(path)
	for _, e := range w.Extensions {
		if ext == e {
			matchesExtension = true
			break
		}
	}

	// Check if the path matches any include patterns
	matchesIncludePattern := false
	for _, pattern := range w.IncludePatterns {
		if pattern.MatchString(path) {
			matchesIncludePattern = true
			break
		}
	}

	if matchesExtension || matchesIncludePattern {
		fileData, found := w.FileData[path]
		if !found {
			w.FileData[path] = &FileData{
				Path:    path,
				ModTime: info.ModTime(),
			}
			log.Printf("Found new file: %s\n", path)
			w.ChangeDetected = true
		} else if fileData.ModTime != info.ModTime() {
			fileData.ModTime = info.ModTime()
			log.Printf("File changed: %s\n", path)
			w.ChangeDetected = true
		}
	}
	return nil
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

func (w *Watcher) Start() {
	ticker := time.NewTicker(w.RefreshRate * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			if w.runCmd != nil {
				_ = w.runCmd.Process.Kill()
			}
			return

		case <-ticker.C:
			for _, dir := range w.IncludeDirs {
				err := filepath.Walk(dir, w.visitFile)
				if err != nil {
					log.Printf("Error walking the path %s: %v\n", dir, err)
				}
			}

			// Check for deleted files
			for path := range w.FileData {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					delete(w.FileData, path)
					log.Printf("File deleted: %s\n", path)
					w.ChangeDetected = true
				}
			}

			if w.ChangeDetected {
				w.executeBuildCommand(w.BuildCommand)

				if w.RunCommand != nil {
					if w.runCmd != nil {
						_ = w.runCmd.Process.Kill()
					}
					runCmd, err := w.executeRunCommand(*w.RunCommand)
					if err != nil {
						log.Printf("Error executing run command: %v\n", err)
						continue
					}
					w.runCmd = runCmd
				}

				w.ChangeDetected = false
			}
		}
	}
}

func (w *Watcher) Stop() {
	close(w.stop)
}
