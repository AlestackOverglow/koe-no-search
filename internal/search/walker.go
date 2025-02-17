package search

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"runtime"
)

var skipDirCache struct {
	sync.RWMutex
	paths map[string]bool
}

func init() {
	skipDirCache.paths = make(map[string]bool, 1000)
}

// shouldSkipDirectory checks if the directory should be skipped
func shouldSkipDirectory(dir string, opts SearchOptions) bool {
	skipDirCache.RLock()
	if skip, ok := skipDirCache.paths[dir]; ok {
		skipDirCache.RUnlock()
		return skip
	}
	skipDirCache.RUnlock()

	skip := false
	base := filepath.Base(dir)

	if opts.ExcludeHidden && strings.HasPrefix(base, ".") {
		skip = true
	} else if skipDirs[filepath.Base(dir)] {
		skip = true
	} else {
		for _, excludeDir := range opts.ExcludeDirs {
			if strings.HasPrefix(dir, excludeDir) {
				skip = true
				break
			}
		}
	}

	skipDirCache.Lock()
	skipDirCache.paths[dir] = skip
	skipDirCache.Unlock()

	if skip {
		logDebug("Skipping directory: %s", dir)
	}
	return skip
}

// walkDirectoryOptimized processes a directory and its subdirectories with optimizations
func walkDirectoryOptimized(dir string, paths chan<- string, opts SearchOptions) {
	if shouldSkipDirectory(dir, opts) {
		return
	}

	const batchSize = 1000
	batch := make([]string, 0, batchSize)
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		logError("Failed to walk directory %s: %v", dir, err)
		return
	}
	
	dirs := make([]string, 0, len(entries))
	
	for _, entry := range entries {
		select {
		case <-opts.StopChan:
			return
		default:
			path := filepath.Join(dir, entry.Name())
			
			if entry.IsDir() {
				dirs = append(dirs, path)
			} else if shouldProcessFile(path, opts) {
				batch = append(batch, path)
				if len(batch) >= batchSize {
					sendBatch(batch, paths, opts.StopChan)
					batch = make([]string, 0, batchSize)
				}
			}
		}
	}
	
	if len(batch) > 0 {
		sendBatch(batch, paths, opts.StopChan)
	}
	
	if len(dirs) > 0 {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, runtime.NumCPU())
		
		for _, subdir := range dirs {
			select {
			case <-opts.StopChan:
				return
			case semaphore <- struct{}{}:
				wg.Add(1)
				go func(d string) {
					defer func() {
						<-semaphore
						wg.Done()
					}()
					walkDirectoryOptimized(d, paths, opts)
				}(subdir)
			}
		}
		wg.Wait()
	}
}

// sendBatch отправляет пакет файлов в канал
func sendBatch(batch []string, paths chan<- string, stopChan chan struct{}) {
	for _, path := range batch {
		select {
		case paths <- path:
		case <-stopChan:
			return
		}
	}
}

// processDirectoryEntry processes a cached directory entry
func processDirectoryEntry(entry *DirEntry, paths chan<- string, opts SearchOptions) {
	if entry == nil {
		logError("Received nil directory entry")
		return
	}
	
	// Process children recursively
	for _, child := range entry.children {
		select {
		case <-opts.StopChan:
			logDebug("Search stopped while processing cached entries")
			return
		default:
			if child.entry.IsDir() {
				processDirectoryEntry(child, paths, opts)
			} else {
				path := filepath.Join(entry.entry.Name(), child.entry.Name())
				if shouldProcessFile(path, opts) {
					select {
					case paths <- path:
						logDebug("Added cached file to processing queue: %s", path)
					case <-opts.StopChan:
						logDebug("Search stopped while adding cached file: %s", path)
						return
					}
				}
			}
		}
	}
}

// getPathPriority determines the priority of a file
func getPathPriority(path string, opts SearchOptions) int {
	for _, dir := range opts.PriorityDirs {
		if strings.HasPrefix(path, dir) {
			return 2
		}
	}
	for _, dir := range opts.LowPriorityDirs {
		if strings.HasPrefix(path, dir) {
			return 0
		}
	}
	return 1
}

// sendToPriorityQueue sends a file to the appropriate queue
func sendToPriorityQueue(path string, metadata *FileMetadata, opts SearchOptions) {
	priority := getPathPriority(path, opts)
	switch priority {
	case 2:
		highPriorityPaths <- path
	case 1:
		normalPriorityPaths <- path
	case 0:
		lowPriorityPaths <- path
	}
} 