package search

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"github.com/cespare/xxhash"
)

// Structures for pattern matching
type compiledPatterns struct {
	simplePatterns [][]byte
	extensions     [][]byte
	ignoreCase    bool
	// Добавляем кэш для часто используемых шаблонов
	commonPatterns map[string]struct{}
}

// Cache for compiled patterns to avoid recompilation
var patternCache struct {
	sync.RWMutex
	lowerCache map[string][]byte
}

func init() {
	patternCache.lowerCache = make(map[string][]byte, 1000)
}

// preparePatterns pre-compiles patterns for faster matching
func preparePatterns(opts SearchOptions) compiledPatterns {
	simplePatterns := make([][]byte, 0, len(opts.Patterns))
	extensions := make([][]byte, 0, len(opts.Extensions))
	commonPatterns := make(map[string]struct{}, len(opts.Patterns))
	
	// Pre-process extensions once
	for _, ext := range opts.Extensions {
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if opts.IgnoreCase {
			ext = strings.ToLower(ext)
		}
		extensions = append(extensions, []byte(ext))
		commonPatterns[ext] = struct{}{}
	}
	
	// Process patterns
	for _, pat := range opts.Patterns {
		if pat == "" {
			continue
		}
		
		if opts.IgnoreCase {
			pat = strings.ToLower(pat)
		}
		simplePatterns = append(simplePatterns, []byte(pat))
		commonPatterns[pat] = struct{}{}
	}
	
	return compiledPatterns{
		simplePatterns: simplePatterns,
		extensions:     extensions,
		ignoreCase:    opts.IgnoreCase,
		commonPatterns: commonPatterns,
	}
}

// matchesPatterns checks if a file matches the compiled patterns
func matchesPatterns(path string, patterns compiledPatterns, _ bool) bool {
	if len(patterns.simplePatterns) == 0 && len(patterns.extensions) == 0 {
		return true
	}

	// Get filename and extension once
	filename := filepath.Base(path)
	ext := filepath.Ext(path)
	
	// Быстрая проверка через map для частых шаблонов
	if patterns.ignoreCase {
		if ext != "" {
			if _, ok := patterns.commonPatterns[strings.ToLower(ext)]; ok {
				return true
			}
		}
		lowerFilename := strings.ToLower(filename)
		if _, ok := patterns.commonPatterns[lowerFilename]; ok {
			return true
		}
	} else {
		if ext != "" {
			if _, ok := patterns.commonPatterns[ext]; ok {
				return true
			}
		}
		if _, ok := patterns.commonPatterns[filename]; ok {
			return true
		}
	}
	
	// Проверка расширений
	if len(patterns.extensions) > 0 && ext != "" {
		extBytes := []byte(ext)
		if patterns.ignoreCase {
			extBytes = bytes.ToLower(extBytes)
		}
		for _, e := range patterns.extensions {
			if bytes.Equal(extBytes, e) {
				return true
			}
		}
	}
	
	// Проверка паттернов
	filenameBytes := []byte(filename)
	if patterns.ignoreCase {
		filenameBytes = bytes.ToLower(filenameBytes)
	}
	
	for _, pattern := range patterns.simplePatterns {
		if len(pattern) <= len(filenameBytes) {
			if bytes.Contains(filenameBytes, pattern) {
				return true
			}
		}
	}
	
	return false
}

// shouldProcessFile performs quick checks before more expensive operations
func shouldProcessFile(path string, opts SearchOptions) bool {
	if len(opts.Patterns) == 0 {
		return true
	}
	
	filename := filepath.Base(path)
	
	if opts.IgnoreCase {
		lowerFilename := strings.ToLower(filename)
		for _, pattern := range opts.Patterns {
			if pattern == "" {
				continue
			}
			if strings.Contains(lowerFilename, strings.ToLower(pattern)) {
				return true
			}
		}
	} else {
		for _, pattern := range opts.Patterns {
			if pattern != "" && strings.Contains(filename, pattern) {
				return true
			}
		}
	}
	
	return false
}

// matchesFileConstraints checks if file matches size and age constraints
func matchesFileConstraints(_ os.FileInfo, _ SearchOptions) bool {
	return true
}

// processByMMap processes a file using memory mapping
func processByMMap(path string, info os.FileInfo, _ compiledPatterns, _ SearchOptions, processor *resultProcessor) error {
	processor.add(SearchResult{
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
	})
	return nil
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