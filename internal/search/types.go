package search

import (
	"io/fs"
	"os"
	"sync"
	"time"
	"regexp"
)

// SearchResult represents a single found file
type SearchResult struct {
	Path      string
	Size      int64
	Mode      os.FileMode
	ModTime   time.Time
	Hash      uint64    // Quick hash for duplicate detection
	Error     error     // Error if occurred during processing
}

// SearchOptions contains search parameters
type SearchOptions struct {
	RootDirs         []string        // List of root directories to search
	Pattern          string
	Extension        string
	MaxWorkers       int
	IgnoreCase       bool
	BufferSize       int             // Channel buffer size
	MinSize          int64           // Minimum file size
	MaxSize          int64           // Maximum file size
	MinAge          time.Duration    // Minimum file age
	MaxAge          time.Duration    // Maximum file age
	ExcludeHidden    bool           // Exclude hidden files and directories
	FollowSymlinks   bool           // Follow symbolic links
	DeduplicateFiles bool           // Remove duplicate files
	BatchSize        int            // Number of files to process in batch
	UseMMap          bool           // Use memory mapping for large files
	MinMMapSize      int64          // Minimum file size for using mmap
	UsePreIndexing   bool           // Use pre-indexing
	IndexPath        string         // Path for saving index
	PriorityDirs     []string       // Directories for priority search
	LowPriorityDirs  []string       // Directories for low priority search
	StopChan         chan struct{}  // Channel for stopping the search
}

// FileMetadata stores file metadata for quick comparison
type FileMetadata struct {
	Size     int64
	ModTime  time.Time
	DeviceID uint64
	InodeID  uint64
}

// DirEntry represents a cached directory entry with additional metadata
type DirEntry struct {
	entry    fs.DirEntry
	modTime  time.Time
	size     int64
	children []*DirEntry
}

// DirStats stores directory statistics
type DirStats struct {
	FileCount     int
	TotalSize     int64
	LastModified  time.Time
	CommonExts    map[string]int
	UpdateCount   int64
}

// FileIndex stores pre-built file information
type FileIndex struct {
	Files     map[string]*FileMetadata
	DirStats  map[string]*DirStats
	LastBuild time.Time
	sync.RWMutex
}

// compiledPatterns holds pre-compiled patterns for faster matching
type compiledPatterns struct {
	pattern   *regexp.Regexp
	extension string
}

// BatchProcessor processes files in batches for better performance
type BatchProcessor struct {
	batch    []string
	size     int
	callback func([]string)
} 