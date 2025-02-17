# Koe no Search - API Documentation

This document describes the public API of the Koe no Search library, which you can use to integrate file search capabilities into your Go applications.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Types](#core-types)
- [Search Functions](#search-functions)
- [Error Handling](#error-handling)
- [Performance Features](#performance-features)
- [Logging System](#logging-system)
- [GUI Integration](#gui-integration)
- [File Operations](#file-operations)
- [Result Handling](#result-handling)
- [Platform Support](#platform-support)
- [Testing](#testing)
- [Security Considerations](#security-considerations)

## Installation

```bash
go get github.com/AlestackOverglow/koe-no-search
```

### Requirements
- Go 1.21 or later
- For GUI functionality:
  - Windows: GCC (MinGW-w64)
  - Linux: X11 and XCB development libraries
  - macOS: Xcode Command Line Tools

## Quick Start

Here's a simple example of how to use the search API:

```go
package main

import (
    "fmt"
    "runtime"
    "time"
    "filesearch/internal/search"
)

func main() {
    // Create stop channel for graceful cancellation
    stopChan := make(chan struct{})
    
    // Configure search options
    opts := search.SearchOptions{
        RootDirs:    []string{"/path/to/search"},
        Patterns:    []string{"*.txt", "*.doc"},
        Extensions:  []string{"txt", "doc"},
        MaxWorkers:  runtime.NumCPU(),
        IgnoreCase:  true,
        BufferSize:  2000,
        StopChan:    stopChan,
        
        // Performance features are disabled by default
        DeduplicateFiles: false,
        FollowSymlinks:   false,
        UseMMap:          false,
        ExcludeHidden:    false,
        MinSize:          0,
        MaxSize:          0,
        MinAge:           0,
        MaxAge:           0,
    }

    // Start search
    startTime := time.Now()
    results := search.Search(opts)
    
    // Process results
    count := 0
    errors := 0
    
    for result := range results {
        if result.Error != nil {
            fmt.Printf("Error: %v\n", result.Error)
            errors++
            continue
        }
        count++
        fmt.Printf("Found: %s (Size: %d bytes)\n", result.Path, result.Size)
    }
    
    // Print summary
    duration := time.Since(startTime)
    fmt.Printf("Search completed in %v. Found %d files, %d errors\n",
        duration, count, errors)
}
```

## Core Types

### SearchOptions

```go
type SearchOptions struct {
    RootDirs         []string        // List of root directories to search
    Patterns         []string        // List of search patterns
    Extensions       []string        // List of file extensions
    MaxWorkers       int            // Number of concurrent workers
    IgnoreCase       bool           // Case-insensitive search
    BufferSize       int            // Channel buffer size for results
    MinSize          int64          // Minimum file size
    MaxSize          int64          // Maximum file size
    MinAge          time.Duration   // Minimum file age
    MaxAge          time.Duration   // Maximum file age
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
```

#### Default Values
- `MaxWorkers`: runtime.NumCPU()
- `BufferSize`: 1000
- `BatchSize`: 100
- `MinMMapSize`: 100MB
- All boolean options default to false
- File filtering options (MinSize, MaxSize, MinAge, MaxAge) default to 0

### SearchResult

```go
type SearchResult struct {
    Path      string      // Full path to the file
    Size      int64       // File size in bytes
    Mode      os.FileMode // File mode and permissions
    ModTime   time.Time   // Last modification time
    Error     error       // Error if occurred during processing
}
```

### FileListItem
```go
type FileListItem struct {
    Path string    // Full path to the file
    Size int64     // File size in bytes
}
```

### ResultBuffer
```go
type ResultBuffer struct {
    // Internal buffer for search results
    items      []FileListItem
    foundFiles *[]FileListItem
    list       *VirtualFileList
    
    // Update thresholds
    minUpdateInterval time.Duration
    batchSize        int
}
```

### Pattern Matching

#### Pattern Cache
```go
var patternCache struct {
    sync.RWMutex
    lowerCache map[string][]byte
}
```

The pattern cache stores pre-compiled patterns for better performance:
- Case-folded patterns for case-insensitive search
- Common patterns shared between searches
- Efficient byte slice storage

#### Pattern Compilation
```go
type compiledPatterns struct {
    simplePatterns [][]byte       // Pre-compiled patterns
    extensions     [][]byte       // Pre-compiled extensions
    ignoreCase     bool          // Case sensitivity flag
    commonPatterns map[string]struct{} // Cache for frequent patterns
}
```

### File Operations

#### Operation Types
```go
type FileOperation int

const (
    NoOperation FileOperation = iota
    CopyFiles
    MoveFiles
    DeleteFiles
)
```

#### Operation Options
```go
type FileOperationOptions struct {
    Operation       FileOperation
    TargetDir       string
    ConflictPolicy  ConflictResolutionPolicy
}

type ConflictResolutionPolicy int

const (
    Skip ConflictResolutionPolicy = iota
    Overwrite
    Rename
)
```

#### Processor Configuration
```go
type ProcessorOptions struct {
    Workers          int           // Number of concurrent workers
    MaxQueueSize     int           // Maximum queued operations
    ThrottleInterval time.Duration // Interval between operations
}
```

### Performance Features

#### Memory Management and Pattern Matching
```go
// Global buffer pools
var (
    // Regular buffer pool (32KB)
    bufferPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 32*1024)
        },
    }
    
    // Large file buffer pool (1MB)
    mmapPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 1024*1024)
        },
    }
)

// Pattern matching optimization
type compiledPatterns struct {
    simplePatterns [][]byte           // Compiled search patterns
    extensions     [][]byte           // Compiled extensions
    ignoreCase     bool              // Case sensitivity flag
    commonPatterns map[string]struct{} // Frequently used patterns
}

// Pattern cache
var patternCache struct {
    sync.RWMutex
    lowerCache map[string][]byte
}

// Features:
// - Pattern pre-compilation
// - Common pattern caching
// - Case folding optimization
// - Extension-first matching
// - Regular expression support
// - Multiple pattern groups
// - Parallel pattern matching
```

### Directory Walking and Processing
```go
// Optimized directory traversal
func walkDirectoryOptimized(dir string, paths chan<- string, opts SearchOptions) {
    // Features:
    // - Concurrent subdirectory processing
    // - Directory entry batching
    // - Memory-efficient traversal
    // - Priority-based processing
}

// Skip common directories
var skipDirs = map[string]bool{
    "node_modules": true,
    ".git":        true,
    "target":      true,
    "dist":        true,
}

// Priority processing
var (
    highPriorityPaths   = make(chan string, 10000)
    normalPriorityPaths = make(chan string, 10000)
    lowPriorityPaths    = make(chan string, 10000)
)
```

## Logging System

### Logging System Implementation

#### Logger Structure
```go
// Logger configuration
const (
    maxLogSize       = 10 * 1024 * 1024  // 10MB
    logBufferSize    = 32 * 1024         // 32KB
    maxLogRotations  = 5
)

type Logger struct {
    mu       sync.RWMutex
    writer   *bufio.Writer
    file     *os.File
    buffer   []byte
    disabled bool
}

var (
    globalLogger  *Logger
    loggerOnce   sync.Once
    loggerBuffer = make(chan string, 1000)  // Buffered channel for logs
)
```

#### Logging Functions
```go
// Log levels
type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARNING
    ERROR
)

// Logging functions
func logDebug(format string, args ...interface{})
func logInfo(format string, args ...interface{})
func logWarning(format string, args ...interface{})
func logError(format string, args ...interface{})

// Example usage:
logInfo("Processing directory: %s", dir)
logError("Failed to access file: %v", err)
```

#### Log File Management
```go
// Initialize logger
func initLogger() {
    // Features:
    // - Automatic log directory creation
    // - Log file rotation
    // - Buffer management
    // - Error recovery
}

// Rotate log files
func rotateLogFile(logPath string) {
    // Rotation features:
    // - Size-based rotation
    // - Multiple backup files
    // - Atomic rename operations
}

// Asynchronous log processing
func processLogs() {
    // Processing features:
    // - Non-blocking writes
    // - Batch processing
    // - Error handling
}
```

## GUI Integration

### Virtual List
```go
// Create virtual list for efficient result display
vlist := NewVirtualFileList(foundFiles)

// Configure list options
vlist.SetMaxCacheSize(10000)
vlist.SetVisibleBuffer(100)

// Set selection callback
vlist.SetOnSelected(func(id int) {
    if id < len(*foundFiles) {
        path := (*foundFiles)[id].Path
        explorer.ShowInExplorer(path)
    }
})
```

### Progress Tracking
```go
// Create and configure progress bar
progress := widget.NewProgressBarInfinite()
progress.Show()
defer progress.Hide()

// Update search time
duration := time.Since(startTime)
searchTimeLabel.SetText(fmt.Sprintf("Search completed in %v", duration))
```

### Theme Customization
```go
// Custom theme for better visibility
type customTheme struct {
    base fyne.Theme
}

func (t *customTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
    if n == theme.ColorNameButton {
        return color.NRGBA{R: 145, G: 85, B: 95, A: 180}
    }
    return t.base.Color(n, v)
}
```

### Layout Management
```go
// Create split container
split := container.NewHSplit(
    scrolledInputs,
    container.NewScroll(vlist.AsListWidget()),
)
split.SetOffset(0.3) // Left part takes 30% of window width

// Add background
if len(search.BackgroundData) > 0 {
    bgImage := canvas.NewImageFromResource(bgResource)
    bgImage.FillMode = canvas.ImageFillStretch
    bgImage.Translucency = 0.9
}
```

## File Operations

### Core Operations
```go
// Operation types
const (
    OperationCopy = iota
    OperationMove
    OperationDelete
)

// Basic operations
func CopyFiles(files []string, targetDir string, progressCallback func(current, total int, path string)) error
func MoveFiles(files []string, targetDir string, progressCallback func(current, total int, path string)) error
func DeleteFiles(files []string, progressCallback func(current, total int, path string)) error

// Operation processor
type ProcessorOptions struct {
    Workers          int
    MaxQueueSize     int
    ThrottleInterval time.Duration
}

// Create processor
processor := NewFileOperationProcessor(ProcessorOptions{
    Workers:          runtime.NumCPU() / 2,
    MaxQueueSize:     1000,
    ThrottleInterval: 100 * time.Millisecond,
})
```

### Operation Safety
```go
// Validate operation
func validateOperation(files []string, targetDir string) error {
    for _, file := range files {
        if _, err := os.Stat(file); err != nil {
            return fmt.Errorf("source file not found: %s", file)
        }
        
        targetPath := filepath.Join(targetDir, filepath.Base(file))
        if _, err := os.Stat(targetPath); err == nil {
            return fmt.Errorf("target file already exists: %s", targetPath)
        }
    }
    return nil
}

// Operation rollback
type OperationLog struct {
    SourcePath string
    TargetPath string
    Operation  int
}

func rollbackOperations(logs []OperationLog) error {
    for i := len(logs) - 1; i >= 0; i-- {
        log := logs[i]
        switch log.Operation {
        case OperationCopy:
            os.Remove(log.TargetPath)
        case OperationMove:
            os.Rename(log.TargetPath, log.SourcePath)
        }
    }
    return nil
}
```

## Result Handling

### Virtual List Configuration
```go
// Create virtual list with optimized settings
vlist := NewVirtualFileList(&foundFiles)
vlist.maxCacheSize = 10000    // Increased cache size
vlist.visibleBuffer = 100     // Increased buffer zone

// Configure update behavior
vlist.updateInterval = 50 * time.Millisecond
vlist.updateBatchSize = 100
```

### Result Buffering
```go
// Create result buffer with optimal settings
buffer := NewResultBuffer(&foundFiles, vlist)
buffer.minUpdateInterval = 100 * time.Millisecond
buffer.batchSize = 1000

// Process results in batches
for result := range results {
    buffer.Add(FileListItem{
        Path: result.Path,
        Size: result.Size,
    })
}
```

### Memory Management
```go
// Configure buffer sizes based on available memory
if runtime.GOOS == "windows" {
    buffer.batchSize = 2000        // Larger batches on Windows
    vlist.maxCacheSize = 15000     // Larger cache on Windows
}

// Clear cache when needed
vlist.cache.Clear()
```

### Result Filtering
```go
// Filter results by criteria
func filterResults(results []FileListItem, criteria FilterCriteria) []FileListItem {
    filtered := make([]FileListItem, 0)
    for _, item := range results {
        if criteria.MinSize > 0 && item.Size < criteria.MinSize {
            continue
        }
        if criteria.MaxSize > 0 && item.Size > criteria.MaxSize {
            continue
        }
        filtered = append(filtered, item)
    }
    return filtered
}
```

### Result Sorting
```go
// Sort results by different criteria
type SortBy int

const (
    SortByName SortBy = iota
    SortBySize
    SortByPath
)

func sortResults(results []FileListItem, by SortBy) {
    sort.Slice(results, func(i, j int) bool {
        switch by {
        case SortByName:
            return filepath.Base(results[i].Path) < filepath.Base(results[j].Path)
        case SortBySize:
            return results[i].Size < results[j].Size
        case SortByPath:
            return results[i].Path < results[j].Path
        default:
            return false
        }
    })
}
```

## Platform Support

### Windows
- File paths are case-insensitive by default
- UNC paths are supported
- System directory exclusions

### Linux/Unix
- File paths are case-sensitive by default
- Symbolic link support
- Hidden file handling (.dotfiles)

### macOS
- File paths are case-insensitive by default
- Resource forks are ignored
- Bundle contents are searchable

## Testing

### Unit Tests
```go
func TestSearch(t *testing.T) {
    // Test setup
    tempDir := t.TempDir()
    createTestFiles(tempDir)
    
    // Test search
    opts := SearchOptions{
        RootDirs: []string{tempDir},
        Patterns: []string{"*.txt"},
    }
    
    results := Search(opts)
    validateResults(t, results)
}
```

### Benchmarks
```go
func BenchmarkSearch(b *testing.B) {
    // Benchmark setup
    opts := SearchOptions{
        RootDirs: []string{"testdata"},
        Patterns: []string{"*.txt"},
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        results := Search(opts)
        for range results {
            // Process results
        }
    }
}
```

### Integration Tests
```go
func TestSearchIntegration(t *testing.T) {
    // Test with real file system
    opts := SearchOptions{
        RootDirs:    []string{"/test/data"},
        Patterns:    []string{"*.txt"},
        MaxWorkers:  2,
        BufferSize:  1000,
    }
    
    results := Search(opts)
    validateIntegrationResults(t, results)
}
```

## Security Considerations

### File Access Control
```go
// Check file permissions
func checkFilePermissions(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    
    // Check read permissions
    if info.Mode().Perm()&0400 == 0 {
        return ErrAccessDenied
    }
    
    return nil
}
```

### Safe File Operations
```go
// Safe file copy with validation
func SafeCopyFile(src, dst string) error {
    // Validate paths
    if err := validatePaths(src, dst); err != nil {
        return err
    }
    
    // Copy with temp file
    tmpDst := dst + ".tmp"
    if err := copyFile(src, tmpDst); err != nil {
        os.Remove(tmpDst)
        return err
    }
    
    return os.Rename(tmpDst, dst)
}
```

### Path Sanitization
```go
// Sanitize file path
func SanitizePath(path string) string {
    // Clean path
    path = filepath.Clean(path)
    
    // Convert to absolute path
    if !filepath.IsAbs(path) {
        if abs, err := filepath.Abs(path); err == nil {
            path = abs
        }
    }
    
    return path
}
```

## Error Handling

### Error Types and Recovery
```go
// Define custom errors
type SearchError struct {
    Path string
    Op   string
    Err  error
}

func (e *SearchError) Error() string {
    return fmt.Sprintf("%s: %s: %v", e.Op, e.Path, e.Err)
}

// Common error types
var (
    ErrAccessDenied   = errors.New("access denied")
    ErrNotFound       = errors.New("file not found")
    ErrInvalidPattern = errors.New("invalid search pattern")
    ErrCancelled      = errors.New("search cancelled")
)

// Error aggregation
type ErrorList struct {
    Errors []error
    mu     sync.Mutex
}

func (e *ErrorList) Add(err error) {
    e.mu.Lock()
    e.Errors = append(e.Errors, err)
    e.mu.Unlock()
}

// Error recovery and safety
func recoverSearch(results chan<- SearchResult) {
    if r := recover(); r != nil {
        results <- SearchResult{
            Error: fmt.Errorf("search panic: %v", r),
        }
        close(results)
    }
}

// Safe file operations with validation
func SafeCopyFile(src, dst string) error {
    // Validate paths
    if err := validatePaths(src, dst); err != nil {
        return err
    }
    
    // Copy with temp file
    tmpDst := dst + ".tmp"
    if err := copyFile(src, tmpDst); err != nil {
        os.Remove(tmpDst)
        return err
    }
    
    return os.Rename(tmpDst, dst)
}
```

## Performance Features

### Memory Management and Pattern Matching
```go
// Global buffer pools
var (
    // Regular buffer pool (32KB)
    bufferPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 32*1024)
        },
    }
    
    // Large file buffer pool (1MB)
    mmapPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 1024*1024)
        },
    }
)

// Pattern matching optimization
type compiledPatterns struct {
    simplePatterns [][]byte           // Compiled search patterns
    extensions     [][]byte           // Compiled extensions
    ignoreCase     bool              // Case sensitivity flag
    commonPatterns map[string]struct{} // Frequently used patterns
}

// Pattern cache
var patternCache struct {
    sync.RWMutex
    lowerCache map[string][]byte
}

// Features:
// - Pattern pre-compilation
// - Common pattern caching
// - Case folding optimization
// - Extension-first matching
// - Regular expression support
// - Multiple pattern groups
// - Parallel pattern matching
```

### Directory Walking and Processing
```go
// Optimized directory traversal
func walkDirectoryOptimized(dir string, paths chan<- string, opts SearchOptions) {
    // Features:
    // - Concurrent subdirectory processing
    // - Directory entry batching
    // - Memory-efficient traversal
    // - Priority-based processing
}

// Skip common directories
var skipDirs = map[string]bool{
    "node_modules": true,
    ".git":        true,
    "target":      true,
    "dist":        true,
}

// Priority processing
var (
    highPriorityPaths   = make(chan string, 10000)
    normalPriorityPaths = make(chan string, 10000)
    lowPriorityPaths    = make(chan string, 10000)
)
```

