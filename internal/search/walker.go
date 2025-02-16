package search

import (
	"os"
	"path/filepath"
	"strings"
)

// walkDirectoryOptimized processes a directory and its subdirectories with optimizations
func walkDirectoryOptimized(dir string, paths chan<- string, opts SearchOptions) {
	logDebug("Starting directory walk: %s", dir)
	
	// Skip hidden directories
	if opts.ExcludeHidden {
		base := filepath.Base(dir)
		if strings.HasPrefix(base, ".") {
			logDebug("Skipping hidden directory: %s", dir)
			return
		}
	}

	if skipDirs[filepath.Base(dir)] {
		logDebug("Skipping excluded directory: %s", dir)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		logError("Failed to read directory %s: %v", dir, err)
		return
	}
	
	for _, entry := range entries {
		select {
		case <-opts.StopChan:
			logDebug("Search stopped while processing directory: %s", dir)
			return
		default:
			path := filepath.Join(dir, entry.Name())
			
			if entry.IsDir() {
				if !skipDirs[entry.Name()] {
					walkDirectoryOptimized(path, paths, opts)
				} else {
					logDebug("Skipping excluded subdirectory: %s", path)
				}
			} else {
				if shouldProcessFile(path, opts) {
					select {
					case paths <- path:
						logDebug("Added file to processing queue: %s", path)
					case <-opts.StopChan:
						logDebug("Search stopped while adding file: %s", path)
						return
					}
				}
			}
		}
	}
	
	logDebug("Finished processing directory: %s", dir)
}

// processDirectoryEntry processes a cached directory entry
func processDirectoryEntry(entry *DirEntry, paths chan<- string, opts SearchOptions) {
	if entry == nil {
		logWarning("Received nil directory entry")
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