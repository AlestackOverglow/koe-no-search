package search

import (
	"fmt"
	"os"
)

// processRegularFile processes a regular file
func processRegularFile(path string, info os.FileInfo, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor, buf []byte) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		logDebug("File no longer exists or inaccessible: %s: %v", path, err)
		return
	}

	if matchesPatterns(path, patterns, opts.IgnoreCase) &&
		matchesFileConstraints(info, opts) {
		
		var hash uint64
		var err error
		
		// Safe hash calculation
		func() {
			defer func() {
				if r := recover(); r != nil {
					logError("Panic while calculating hash for %s: %v", path, r)
					err = fmt.Errorf("hash calculation failed: %v", r)
				}
			}()
			hash = calculateQuickHash(path, info, buf)
		}()
		
		result := SearchResult{
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    hash,
			Error:   err,
		}
		
		// Check file accessibility
		if f, err := os.Open(path); err != nil {
			logError("Failed to access file: %s: %v", path, err)
			result.Error = err
		} else {
			f.Close()
			logDebug("Successfully processed file: %s", path)
		}
		
		processor.add(result)
	}
} 