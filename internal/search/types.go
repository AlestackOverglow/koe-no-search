package search

import (
	"io/fs"
	"os"
	"sync"
	"time"
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
	Patterns         []string        // List of search patterns
	Extensions       []string        // List of file extensions
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
	FileOp           FileOperationOptions
	ExcludeDirs      []string       // Directories to exclude from search
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

// BatchProcessor processes files in batches for better performance
type BatchProcessor struct {
	batch    []string
	size     int
	callback func([]string)
}

// FileOperation represents the type of operation to perform on found files
type FileOperation int

const (
	NoOperation FileOperation = iota
	CopyFiles
	MoveFiles
	DeleteFiles
)

// FileOperationOptions contains settings for file operations
type FileOperationOptions struct {
	Operation       FileOperation
	TargetDir      string
	ConflictPolicy ConflictResolutionPolicy
}

// ConflictResolutionPolicy defines how to handle file name conflicts
type ConflictResolutionPolicy int

const (
	Skip ConflictResolutionPolicy = iota
	Overwrite
	Rename
)

// BloomFilter represents a probabilistic set data structure
type BloomFilter struct {
	bits    []bool
	numBits uint
	numHash uint
}

// BloomFilterOptions contains configuration for bloom filters
type BloomFilterOptions struct {
	ExpectedItems uint    // Expected number of items
	FalsePositive float64 // Acceptable false positive rate (0.0 to 1.0)
}

// FileFilterSet contains bloom filters for different file attributes
type FileFilterSet struct {
	Extensions *BloomFilter // Filter for file extensions
	Paths      *BloomFilter // Filter for file paths
	Dirs       *BloomFilter // Filter for processed directories
	mu         sync.RWMutex
} 