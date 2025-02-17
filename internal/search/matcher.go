package search

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/edsrzf/mmap-go"
	"github.com/cespare/xxhash"
)

// Структуры для работы с паттернами
type compiledPatterns struct {
	patterns       []*regexp.Regexp
	simplePatterns []string
	extensions     []string
}

// Cache for compiled patterns to avoid recompilation
var patternCache struct {
	sync.RWMutex
	patterns       map[string]*regexp.Regexp
	simplePatterns map[string]string
	lowerCache     map[string]string // Кэш для преобразования в нижний регистр
}

func init() {
	patternCache.patterns = make(map[string]*regexp.Regexp, 100)
	patternCache.simplePatterns = make(map[string]string, 100)
	patternCache.lowerCache = make(map[string]string, 1000)
}

// getCompiledPattern returns a cached compiled pattern or creates a new one
func getCompiledPattern(pattern string, ignoreCase bool) *regexp.Regexp {
	if ignoreCase {
		pattern = "(?i)" + pattern
	}

	patternCache.RLock()
	if re, ok := patternCache.patterns[pattern]; ok {
		patternCache.RUnlock()
		return re
	}
	patternCache.RUnlock()

	patternCache.Lock()
	defer patternCache.Unlock()

	// Double check after acquiring write lock
	if re, ok := patternCache.patterns[pattern]; ok {
		return re
	}

	re := regexp.MustCompile(pattern)
	patternCache.patterns[pattern] = re
	return re
}

// preparePatterns pre-compiles patterns for faster matching
func preparePatterns(opts SearchOptions) compiledPatterns {
	patterns := make([]*regexp.Regexp, 0, len(opts.Patterns))
	simplePatterns := make([]string, 0, len(opts.Patterns))
	
	// Предварительно сортируем расширения один раз
	extensions := make([]string, 0, len(opts.Extensions))
	if len(opts.Extensions) > 0 {
		seen := make(map[string]bool, len(opts.Extensions))
		for _, ext := range opts.Extensions {
			if ext == "" {
				continue
			}
			if opts.IgnoreCase {
				ext = strings.ToLower(ext)
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if !seen[ext] {
				extensions = append(extensions, ext)
				seen[ext] = true
			}
		}
		if len(extensions) > 1 {
			sort.Strings(extensions)
		}
	}

	// Оптимизированная обработка паттернов
	for _, pat := range opts.Patterns {
		if pat == "" {
			continue
		}
		
		// Используем кэш для простых паттернов
		if !containsRegexChars(pat) {
			var simplePat string
			if opts.IgnoreCase {
				patternCache.RLock()
				if cached, ok := patternCache.lowerCache[pat]; ok {
					simplePat = cached
					patternCache.RUnlock()
				} else {
					patternCache.RUnlock()
					simplePat = strings.ToLower(pat)
					patternCache.Lock()
					patternCache.lowerCache[pat] = simplePat
					patternCache.Unlock()
				}
			} else {
				simplePat = pat
			}
			simplePatterns = append(simplePatterns, simplePat)
			continue
		}
		
		// Оптимизированное получение regex из кэша
		patternStr := regexp.QuoteMeta(pat)
		if opts.IgnoreCase {
			patternStr = "(?i)" + patternStr
		}
		
		var re *regexp.Regexp
		patternCache.RLock()
		re, ok := patternCache.patterns[patternStr]
		patternCache.RUnlock()
		
		if !ok {
			re = regexp.MustCompile(patternStr)
			patternCache.Lock()
			patternCache.patterns[patternStr] = re
			patternCache.Unlock()
		}
		patterns = append(patterns, re)
	}
	
	return compiledPatterns{
		patterns:       patterns,
		simplePatterns: simplePatterns,
		extensions:     extensions,
	}
}

// containsRegexChars проверяет, содержит ли строка специальные символы regex
func containsRegexChars(s string) bool {
	return strings.ContainsAny(s, "^$.*+?()[]{}|\\")
}

// matchesPatterns checks if a file matches the compiled patterns
func matchesPatterns(path string, patterns compiledPatterns, ignoreCase bool) bool {
	if len(patterns.patterns) == 0 && len(patterns.simplePatterns) == 0 && len(patterns.extensions) == 0 {
		return true
	}

	// Быстрая проверка расширений
	if len(patterns.extensions) > 0 {
		ext := filepath.Ext(path)
		if ignoreCase {
			patternCache.RLock()
			if cached, ok := patternCache.lowerCache[ext]; ok {
				ext = cached
				patternCache.RUnlock()
			} else {
				patternCache.RUnlock()
				ext = strings.ToLower(ext)
				patternCache.Lock()
				patternCache.lowerCache[ext] = ext
				patternCache.Unlock()
			}
		}
		
		if len(patterns.extensions) > 10 {
			i := sort.SearchStrings(patterns.extensions, ext)
			if i >= len(patterns.extensions) || patterns.extensions[i] != ext {
				return false
			}
		} else {
			found := false
			for _, e := range patterns.extensions {
				if ext == e {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	filename := filepath.Base(path)
	var lowerFilename string
	if ignoreCase {
		patternCache.RLock()
		if cached, ok := patternCache.lowerCache[filename]; ok {
			lowerFilename = cached
			patternCache.RUnlock()
		} else {
			patternCache.RUnlock()
			lowerFilename = strings.ToLower(filename)
			patternCache.Lock()
			patternCache.lowerCache[filename] = lowerFilename
			patternCache.Unlock()
		}
	}
	
	// Проверка простых паттернов
	if len(patterns.simplePatterns) > 0 {
		for _, pattern := range patterns.simplePatterns {
			if ignoreCase {
				if strings.Contains(lowerFilename, pattern) {
					return true
				}
			} else {
				if strings.Contains(filename, pattern) {
					return true
				}
			}
		}
		if len(patterns.patterns) == 0 {
			return false
		}
	}
	
	// Проверка regex паттернов
	for _, pattern := range patterns.patterns {
		if pattern.MatchString(filename) {
			return true
		}
	}
	
	return false
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