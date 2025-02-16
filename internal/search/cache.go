package search

import (
	"sync"
	"time"
	"os"
	"strings"
	"path/filepath"
)

// Cache stores directory contents for faster subsequent searches
type Cache struct {
	entries     map[string]*DirEntry
	metadata    map[string]FileMetadata
	hashes      map[uint64]bool      // For deduplication
	sync.RWMutex
	lastUpdate  time.Time
	maxAge      time.Duration
}

// newCache creates a new cache
func newCache() *Cache {
	return &Cache{
		entries:  make(map[string]*DirEntry),
		metadata: make(map[string]FileMetadata),
		hashes:   make(map[uint64]bool),
		maxAge:   5 * time.Minute,
	}
}

// Cache methods
func (c *Cache) get(dir string) (*DirEntry, bool) {
	c.RLock()
	defer c.RUnlock()
	
	entry, ok := c.entries[dir]
	if !ok {
		return nil, false
	}
	
	if time.Since(c.lastUpdate) > c.maxAge {
		return nil, false
	}
	
	return entry, true
}

func (c *Cache) set(dir string, entry *DirEntry) {
	c.Lock()
	defer c.Unlock()
	c.entries[dir] = entry
	c.lastUpdate = time.Now()
}

// updateDirStats updates directory statistics
func updateDirStats(dir string, entries []os.DirEntry) {
	stats := &DirStats{
		CommonExts: make(map[string]int),
		LastModified: time.Now(),
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			stats.FileCount++
			if info, err := entry.Info(); err == nil {
				stats.TotalSize += info.Size()
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				stats.CommonExts[ext]++
			}
		}
	}
	
	fileIndex.Lock()
	fileIndex.DirStats[dir] = stats
	fileIndex.Unlock()
} 