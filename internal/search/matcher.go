package search

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"github.com/edsrzf/mmap-go"
	"github.com/cespare/xxhash"
)

// preparePatterns pre-compiles patterns for faster matching
func preparePatterns(opts SearchOptions) compiledPatterns {
	var pattern *regexp.Regexp
	if opts.Pattern != "" {
		patternStr := regexp.QuoteMeta(opts.Pattern)
		if opts.IgnoreCase {
			patternStr = "(?i)" + patternStr
		}
		pattern = regexp.MustCompile(patternStr)
	}
	
	return compiledPatterns{
		pattern:   pattern,
		extension: opts.Extension,
	}
}

// matchesPatterns checks if a file matches the compiled patterns
func matchesPatterns(path string, patterns compiledPatterns, ignoreCase bool) bool {
	filename := filepath.Base(path)
	
	// Check extension if specified
	if patterns.extension != "" {
		ext := patterns.extension
		if ignoreCase {
			ext = strings.ToLower(ext)
			filename = strings.ToLower(filename)
		}
		if !strings.HasSuffix(filename, ext) {
			return false
		}
	}
	
	// Check pattern if specified
	if patterns.pattern != nil {
		return patterns.pattern.MatchString(filename)
	}
	
	return true
}

// matchesFileConstraints checks if file matches size and age constraints
func matchesFileConstraints(info os.FileInfo, opts SearchOptions) bool {
	// Check file size constraints
	if opts.MinSize > 0 && info.Size() < opts.MinSize {
		logDebug("File too small: %s (%d bytes)", info.Name(), info.Size())
		return false
	}
	if opts.MaxSize > 0 && info.Size() > opts.MaxSize {
		logDebug("File too large: %s (%d bytes)", info.Name(), info.Size())
		return false
	}
	
	// Check file age constraints
	age := time.Since(info.ModTime())
	if opts.MinAge > 0 && age < opts.MinAge {
		logDebug("File too new: %s (age: %v)", info.Name(), age)
		return false
	}
	if opts.MaxAge > 0 && age > opts.MaxAge {
		logDebug("File too old: %s (age: %v)", info.Name(), age)
		return false
	}
	
	return true
}

// processByMMap processes a file using memory mapping
func processByMMap(path string, info os.FileInfo, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		logError("Failed to open file for mmap: %s: %v", path, err)
		return err
	}
	defer f.Close()
	
	mmapData, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		logError("Failed to mmap file: %s: %v", path, err)
		return err
	}
	defer mmapData.Unmap()
	
	// Quick content check
	if matchesContent(mmapData, patterns, opts) {
		hash := xxhash.Sum64(mmapData)
		processor.add(SearchResult{
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    hash,
		})
		logDebug("Successfully processed file with mmap: %s", path)
	}
	
	return nil
}

// matchesContent checks if file content matches the pattern
func matchesContent(data []byte, patterns compiledPatterns, opts SearchOptions) bool {
	if patterns.pattern == nil {
		return true
	}
	
	// Check only first N bytes for large files
	maxCheck := 1024 * 1024 // 1MB
	if len(data) > maxCheck {
		data = data[:maxCheck]
	}
	
	return patterns.pattern.Match(data)
}

// shouldProcessFile performs quick checks before more expensive operations
func shouldProcessFile(path string, opts SearchOptions) bool {
	// Check if file is hidden
	if opts.ExcludeHidden {
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			logDebug("Skipping hidden file: %s", path)
			return false
		}
	}
	
	ext := strings.ToLower(filepath.Ext(path))
	if skipExtensions[ext] {
		logDebug("Skipping file with excluded extension: %s", path)
		return false
	}
	
	// Quick pattern check without regex
	if opts.Pattern != "" {
		filename := filepath.Base(path)
		if opts.IgnoreCase {
			filename = strings.ToLower(filename)
			if !strings.Contains(filename, strings.ToLower(opts.Pattern)) {
				logDebug("File does not match pattern (case-insensitive): %s", path)
				return false
			}
		} else {
			if !strings.Contains(filename, opts.Pattern) {
				logDebug("File does not match pattern (case-sensitive): %s", path)
				return false
			}
		}
	}
	
	return true
} 