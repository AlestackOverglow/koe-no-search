package search

import (
	"math"
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
	var patterns []*regexp.Regexp
	
	for _, pat := range opts.Patterns {
		if pat != "" {
			patternStr := regexp.QuoteMeta(pat)
			if opts.IgnoreCase {
				patternStr = "(?i)" + patternStr
			}
			patterns = append(patterns, regexp.MustCompile(patternStr))
		}
	}
	
	// Convert extensions to lowercase if case-insensitive
	extensions := make([]string, 0, len(opts.Extensions))
	for _, ext := range opts.Extensions {
		if ext != "" {
			if opts.IgnoreCase {
				ext = strings.ToLower(ext)
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			extensions = append(extensions, ext)
		}
	}
	
	return compiledPatterns{
		patterns:   patterns,
		extensions: extensions,
	}
}

// matchesPatterns checks if a file matches the compiled patterns
func matchesPatterns(path string, patterns compiledPatterns, ignoreCase bool) bool {
	filename := filepath.Base(path)
	
	// If no patterns and extensions specified, match all files
	if len(patterns.patterns) == 0 && len(patterns.extensions) == 0 {
		return true
	}
	
	// Check extensions if specified
	if len(patterns.extensions) > 0 {
		ext := filepath.Ext(path)
		if ignoreCase {
			ext = strings.ToLower(ext)
		}
		
		matched := false
		for _, e := range patterns.extensions {
			if ext == e {
				matched = true
				break
			}
		}
		
		// If extensions specified but none matched, return false
		if !matched {
			return false
		}
	}
	
	// Check patterns if specified
	if len(patterns.patterns) > 0 {
		matched := false
		for _, pattern := range patterns.patterns {
			if pattern.MatchString(filename) {
				matched = true
				break
			}
		}
		return matched
	}
	
	// If we got here and extensions matched (or none specified), return true
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
	if len(patterns.patterns) == 0 {
		return true
	}
	
	// Check only first N bytes for large files
	maxCheck := 1024 * 1024 // 1MB
	if len(data) > maxCheck {
		data = data[:maxCheck]
	}
	
	for _, pattern := range patterns.patterns {
		if pattern.Match(data) {
			return true
		}
	}
	
	return false
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
	if len(opts.Patterns) > 0 {
		filename := filepath.Base(path)
		if opts.IgnoreCase {
			filename = strings.ToLower(filename)
			matched := false
			for _, pattern := range opts.Patterns {
				if pattern != "" {
					if strings.Contains(filename, strings.ToLower(pattern)) {
						matched = true
						break
					}
				}
			}
			if !matched {
				logDebug("File does not match any patterns (case-insensitive): %s", path)
				return false
			}
		} else {
			matched := false
			for _, pattern := range opts.Patterns {
				if pattern != "" {
					if strings.Contains(filename, pattern) {
						matched = true
						break
					}
				}
			}
			if !matched {
				logDebug("File does not match any patterns (case-sensitive): %s", path)
				return false
			}
		}
	}
	
	return true
}

// NewBloomFilter creates a new Bloom filter with given options
func NewBloomFilter(opts BloomFilterOptions) *BloomFilter {
	// Calculate optimal number of bits and hash functions
	numBits := uint(-float64(opts.ExpectedItems) * math.Log(opts.FalsePositive) / math.Pow(math.Log(2), 2))
	numHash := uint(float64(numBits) / float64(opts.ExpectedItems) * math.Log(2))
	
	return &BloomFilter{
		bits:    make([]bool, numBits),
		numBits: numBits,
		numHash: numHash,
	}
}

// hash generates k different hash values for an item
func (bf *BloomFilter) hash(item string) []uint {
	hashes := make([]uint, bf.numHash)
	h1 := xxhash.Sum64String(item)
	h2 := xxhash.Sum64String(item + "salt")
	
	for i := uint(0); i < bf.numHash; i++ {
		hashes[i] = uint((h1 + uint64(i)*h2) % uint64(bf.numBits))
	}
	
	return hashes
}

// Add adds an item to the Bloom filter
func (bf *BloomFilter) Add(item string) {
	for _, h := range bf.hash(item) {
		bf.bits[h] = true
	}
}

// Contains checks if an item might be in the set
func (bf *BloomFilter) Contains(item string) bool {
	for _, h := range bf.hash(item) {
		if !bf.bits[h] {
			return false
		}
	}
	return true
}

// NewFileFilterSet creates a new set of Bloom filters for file operations
func NewFileFilterSet() *FileFilterSet {
	return &FileFilterSet{
		Extensions: NewBloomFilter(BloomFilterOptions{
			ExpectedItems: 1000,    // Expect 1000 unique extensions
			FalsePositive: 0.001,   // 0.1% false positive rate
		}),
		Paths: NewBloomFilter(BloomFilterOptions{
			ExpectedItems: 100000,  // Expect 100k paths
			FalsePositive: 0.001,   // 0.1% false positive rate
		}),
		Dirs: NewBloomFilter(BloomFilterOptions{
			ExpectedItems: 10000,   // Expect 10k directories
			FalsePositive: 0.001,   // 0.1% false positive rate
		}),
	}
} 