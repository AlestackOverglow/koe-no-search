# Koe no Search - API Documentation

This document describes the public API of the Koe no Search library, which you can use to integrate file search capabilities into your Go applications.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Types](#core-types)
- [Search Functions](#search-functions)
- [Error Handling](#error-handling)
- [Performance Optimization](#performance-optimization)
- [Logging System](#logging-system)
- [GUI Integration](#gui-integration)
- [File Operations](#file-operations)
- [Result Handling](#result-handling)
- [Platform-Specific Notes](#platform-specific-notes)
- [Advanced Features](#advanced-features)
- [Testing](#testing)
- [Security Considerations](#security-considerations)
- [Error Handling Extensions](#error-handling-extensions)

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

#### Buffer Management
```go
// Regular buffer pool (32KB buffers)
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32*1024)
    },
}

// Large file buffer pool (1MB buffers)
var mmapPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024*1024)
    },
}
```

#### Priority Processing
```go
// Priority queues
var (
    highPriorityPaths   = make(chan string, 10000)
    normalPriorityPaths = make(chan string, 10000)
    lowPriorityPaths   = make(chan string, 10000)
)
```

Files are processed according to their priority:
1. High priority (PriorityDirs)
2. Normal priority (default)
3. Low priority (LowPriorityDirs)

#### Directory Skipping
```go
// Common directories to skip
var skipDirs = map[string]bool{
    "node_modules": true,
    ".git": true,
    "target": true,
    "dist": true,
    // ... other system directories
}
```

### Error Handling

#### Result Processing
```go
type SearchResult struct {
    Path      string      // Full path to the file
    Size      int64       // File size in bytes
    Mode      os.FileMode // File mode and permissions
    ModTime   time.Time   // Last modification time
    Hash      uint64      // Quick hash for deduplication
    Error     error       // Error if occurred during processing
}
```

#### Error Recovery
- Panic recovery in worker goroutines
- Safe hash calculation with recovery
- Graceful error reporting through SearchResult

### Platform Support

#### Windows
- Case-insensitive path handling
- UNC path support
- System directory exclusions

#### Unix/Linux
- Case-sensitive path handling
- Symbolic link support
- Hidden file handling (.dotfiles)

### Current Limitations
- No content-based search
- Limited archive file support
- Basic file operation recovery
- Simple pattern matching (no regex)
- Limited network drive optimization

## Search Functions

### Basic Search
```go
// Perform a basic search with default options
results := search.Search(opts)
```

### Result Processing
```go
// Create result buffer for efficient updates
buffer := NewResultBuffer(foundFiles, virtualList)

// Add results to buffer
for result := range results {
    if result.Error != nil {
        continue
    }
    buffer.Add(FileListItem{
        Path: result.Path,
        Size: result.Size,
    })
}

// Flush remaining items
buffer.Flush()
```

### Search Cancellation
```go
// Create cancellable search
stopChan := make(chan struct{})
opts.StopChan = stopChan

// Start search in goroutine
go func() {
    results := search.Search(opts)
    // Process results...
}()

// Cancel search after timeout
time.AfterFunc(5*time.Minute, func() {
    close(stopChan)
})
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
```

### Error Recovery and Safety
```go
// Recover from panics in search operations
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

## Performance Optimization

### Memory Management and Buffering
```go
// Buffer pools for different operations
var (
    // Regular operations buffer pool (32KB)
    bufferPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 32*1024)
        },
    }

    // Memory mapping buffer pool (1MB)
    mmapPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 1024*1024)
        },
    }
)

// Result buffer configuration
type ResultBuffer struct {
    items            []FileListItem
    foundFiles       *[]FileListItem
    list            *VirtualFileList
    minUpdateInterval time.Duration
    batchSize        int
}

// Buffer optimization strategies:
// - Reuse buffers to reduce allocations
// - Size-specific buffer pools
// - Automatic buffer cleanup
// - Adaptive buffer sizes
// - Memory pressure handling
```

### Pattern Matching and Search
```go
// Optimized pattern matching
type compiledPatterns struct {
    simplePatterns [][]byte           // Compiled search patterns
    extensions     [][]byte           // Compiled extensions
    ignoreCase     bool              // Case sensitivity flag
    commonPatterns map[string]struct{} // Frequently used patterns
}

// Pattern matching features:
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
    // - Skip pattern optimization
    // - Memory-efficient traversal
    // - Priority-based processing
}

// Batch processing configuration
const (
    defaultBatchSize = 1000
    maxBatchSize    = 5000
)

// Skip common directories
var skipDirs = map[string]bool{
    "node_modules": true,
    ".git":        true,
    "target":      true,
    "dist":        true,
}
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

### File Operation Types
```go
const (
    OperationCopy = iota
    OperationMove
    OperationDelete
)
```

### Basic Operations
```go
// Copy files
err := fileOp.CopyFiles(selectedFiles, targetDir, progressCallback)

// Move files
err := fileOp.MoveFiles(selectedFiles, targetDir, progressCallback)

// Delete files
err := fileOp.DeleteFiles(selectedFiles, progressCallback)
```

### Progress Tracking
```go
// Progress callback
progressCallback := func(current, total int, path string) {
    progress := float64(current) / float64(total)
    progressBar.SetValue(progress)
    statusLabel.SetText(fmt.Sprintf("Processing: %s", path))
}
```

### Operation Validation
```go
// Validate operation
func validateOperation(files []string, targetDir string) error {
    for _, file := range files {
        // Check if source exists
        if _, err := os.Stat(file); err != nil {
            return fmt.Errorf("source file not found: %s", file)
        }
        
        // Check target path
        targetPath := filepath.Join(targetDir, filepath.Base(file))
        if _, err := os.Stat(targetPath); err == nil {
            return fmt.Errorf("target file already exists: %s", targetPath)
        }
    }
    return nil
}
```

### Error Recovery
```go
// Implement operation rollback
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

## Platform-Specific Notes

### Windows
- File paths are case-insensitive by default
- UNC paths are supported
- Network drives may require additional permissions

### Linux/Unix
- File paths are case-sensitive by default
- Symbolic links are not followed by default
- Hidden files (starting with '.') are included in search

### macOS
- File paths are case-insensitive by default
- Resource forks are ignored
- Bundle contents are searchable

## Advanced Features

### Priority Search
```go
// Configure priority search
opts := SearchOptions{
    PriorityDirs: []string{
        "/important/dir1",
        "/important/dir2",
    },
    LowPriorityDirs: []string{
        "/archive",
        "/temp",
    },
}

// Files are processed in order:
// 1. High priority (PriorityDirs)
// 2. Normal priority (default)
// 3. Low priority (LowPriorityDirs)
```

#### Caching System
```go
// Enable pre-indexing for faster subsequent searches
opts := SearchOptions{
    UsePreIndexing: true,
    IndexPath: "/path/to/index",
}

// Cache stores:
// - Directory contents
// - File metadata
// - Hash values for deduplication
// - Common patterns
```

#### Batch Processing
```go
// Configure batch processing
opts := SearchOptions{
    BatchSize: 1000,  // Process files in batches
    BufferSize: 5000, // Larger buffer for results
}

// Batch processing benefits:
// - Reduced memory allocation
// - Better I/O performance
// - Efficient worker utilization
```

#### Memory Mapping
```go
// Enable memory mapping for large files
opts := SearchOptions{
    UseMMap: true,
    MinMMapSize: 100 * 1024 * 1024, // 100MB
}

// Memory mapping benefits:
// - Faster file access
// - Reduced memory usage
// - Better performance for large files
```

### Performance Optimization

#### Buffer Management
```go
// Global buffer pools
var (
    // Regular buffer pool (32KB buffers)
    bufferPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 32*1024)
        },
    }
    
    // Large file buffer pool (1MB buffers)
    mmapPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 1024*1024)
        },
    }
)

// Buffer optimization strategies:
// - Reuse buffers to reduce allocations
// - Size-specific buffer pools
// - Automatic buffer cleanup
```

#### Pattern Matching
```go
// Optimized pattern matching
type compiledPatterns struct {
    simplePatterns [][]byte
    extensions     [][]byte
    ignoreCase     bool
    commonPatterns map[string]struct{}
}

// Pattern matching features:
// - Pre-compiled patterns
// - Case-sensitivity optimization
// - Common pattern caching
// - Extension-first matching
```

#### Directory Walking
```go
// Optimized directory traversal
func walkDirectoryOptimized(dir string, paths chan<- string, opts SearchOptions) {
    // Features:
    // - Batch processing of entries
    // - Concurrent subdirectory processing
    // - Memory-efficient traversal
    // - Priority-based processing
}

// Configure concurrent processing
opts := SearchOptions{
    MaxWorkers: runtime.NumCPU() * 2,  // More workers for I/O-bound operations
}
```

#### File Operations
```go
// Processor options for file operations
type ProcessorOptions struct {
    Workers          int           // Number of concurrent workers
    MaxQueueSize     int           // Maximum queued operations
    ThrottleInterval time.Duration // Interval between operations
}

// Create file operation processor
processor := NewFileOperationProcessor(ProcessorOptions{
    Workers:          runtime.NumCPU() / 2,
    MaxQueueSize:     1000,
    ThrottleInterval: 100 * time.Millisecond,
})

// Features:
// - Concurrent processing
// - Operation throttling
// - Automatic worker scaling
// - Error recovery
// - Progress tracking
```

### Error Handling

#### Operation Errors
```go
// File operation error handling
func HandleFileOperation(path string, opts FileOperationOptions) error {
    // Pre-operation checks:
    // - File accessibility
    // - Directory permissions
    // - Disk space
    // - File locks
    
    // Error recovery:
    // - Temporary file cleanup
    // - Partial operation rollback
    // - Error logging
}
```

#### Safe File Operations
```go
// Safe file copy with validation
func copyFile(src string, opts FileOperationOptions, srcInfo os.FileInfo) error {
    // Safety features:
    // - Atomic operations
    // - Temporary file usage
    // - Checksum verification
    // - Permission preservation
    // - Error recovery
}

// Conflict resolution
func resolveConflict(path string, policy ConflictResolutionPolicy) string {
    // Resolution strategies:
    // - Skip existing files
    // - Overwrite files
    // - Rename with timestamp
    // - Random suffix generation
}
```

### Logging System

#### Log Levels and Configuration
```go
// Log configuration
const (
    maxLogSize      = 10 * 1024 * 1024 // 10MB
    logBufferSize   = 32 * 1024        // 32KB
    maxLogRotations = 5
)

// Logging features:
// - Asynchronous logging
// - Log rotation
// - Buffer management
// - Level-based filtering
```

#### Performance Logging
```go
// Log performance metrics
type SearchMetrics struct {
    FilesProcessed   int64
    BytesProcessed   int64
    ErrorCount       int64
    ProcessingTime   time.Duration
    MemoryUsage      int64
}

// Metric collection:
// - File processing stats
// - Memory usage tracking
// - Error counting
// - Timing measurements
```

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

## Error Handling Extensions

### Custom Error Types
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

func (e *SearchError) Unwrap() error {
    return e.Err
}
```

### Error Recovery
```go
// Recover from panics
func recoverSearch(results chan<- SearchResult) {
    if r := recover(); r != nil {
        results <- SearchResult{
            Error: fmt.Errorf("search panic: %v", r),
        }
        close(results)
    }
}
```

### Error Aggregation
```go
// Aggregate multiple errors
type ErrorList struct {
    Errors []error
    mu     sync.Mutex
}

func (e *ErrorList) Add(err error) {
    e.mu.Lock()
    e.Errors = append(e.Errors, err)
    e.mu.Unlock()
}

func (e *ErrorList) Error() string {
    return fmt.Sprintf("%d errors occurred", len(e.Errors))
}
```

### Caching System Implementation

#### Cache Structure
```go
// Cache stores directory contents for faster subsequent searches
type Cache struct {
    entries     map[string]*DirEntry    // Cached directory entries
    metadata    map[string]FileMetadata // File metadata cache
    hashes      map[uint64]bool         // Deduplication cache
    lastUpdate  time.Time               // Last cache update time
    maxAge      time.Duration           // Cache expiration time
}

// DirEntry represents a cached directory entry
type DirEntry struct {
    entry    fs.DirEntry    // Original directory entry
    modTime  time.Time      // Last modification time
    size     int64          // Directory size
    children []*DirEntry    // Subdirectories and files
}

// FileMetadata stores file metadata for quick comparison
type FileMetadata struct {
    Size     int64
    ModTime  time.Time
    DeviceID uint64
    InodeID  uint64
}
```

#### Cache Management
```go
// Create new cache instance
cache := newCache()

// Cache configuration
const (
    defaultMaxAge = 5 * time.Minute
    maxCacheSize  = 100000  // Maximum number of entries
)

// Cache operations
func (c *Cache) get(dir string) (*DirEntry, bool)
func (c *Cache) set(dir string, entry *DirEntry)
func (c *Cache) clear()
```

#### Directory Statistics
```go
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
}
```

### Performance Optimizations

#### Bloom Filters
```go
type BloomFilter struct {
    bits    []bool
    numBits uint
    numHash uint
}

type BloomFilterOptions struct {
    ExpectedItems uint    // Expected number of items
    FalsePositive float64 // Acceptable false positive rate
}

type FileFilterSet struct {
    Extensions *BloomFilter // Filter for file extensions
    Paths      *BloomFilter // Filter for file paths
    Dirs       *BloomFilter // Filter for processed directories
    mu         sync.RWMutex
}
```

Bloom filters are used for efficient set membership testing:
- Extensions filter: 1000 items, 0.1% false positive rate
- Paths filter: 100k items, 0.1% false positive rate
- Directories filter: 10k items, 0.1% false positive rate

#### Virtual List
```go
type VirtualFileList struct {
    widget.BaseWidget
    list        *widget.List
    items       *[]FileListItem
    cache       *SimpleCache
    scroller    *container.Scroll
    resultBuf   *ResultBuffer
    
    // Performance settings
    maxCacheSize     int
    visibleBuffer    int
    updateBatchSize  int
    updateInterval   time.Duration
}

type SimpleCache struct {
    items map[int]string
    mutex sync.RWMutex
}
```

The virtual list provides efficient display of large result sets:
- Lazy loading of visible items
- Chunked data storage
- Smooth scrolling support
- Efficient memory usage

#### Result Buffer
```go
type ResultBuffer struct {
    items      []FileListItem
    mu         sync.Mutex
    foundFiles *[]FileListItem
    list       *VirtualFileList
    
    // Chunked storage
    chunks     [][]FileListItem
    chunkSize  int
    totalItems int
    
    // Update thresholds
    minUpdateInterval time.Duration
    batchSize        int
}
```

Result buffer features:
- Chunked storage (50k items per chunk)
- Adaptive update intervals
- Batch processing
- Memory-efficient storage

### Memory Management
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
```

#### Directory Caching
```go
type Cache struct {
    entries     map[string]*DirEntry
    metadata    map[string]FileMetadata
    hashes      map[uint64]bool
    lastUpdate  time.Time
    maxAge      time.Duration
}

type DirEntry struct {
    entry    fs.DirEntry
    modTime  time.Time
    size     int64
    children []*DirEntry
}

type DirStats struct {
    FileCount     int
    TotalSize     int64
    LastModified  time.Time
    CommonExts    map[string]int
    UpdateCount   int64
}
```

### Result Processing

#### Batch Processing
```go
type BatchProcessor struct {
    batch    []string
    size     int
    callback func([]string)
}
```

Batch processing features:
- Configurable batch size
- Memory-efficient processing
- Automatic flushing

#### Result Processor
```go
type resultProcessor struct {
    results  chan<- SearchResult
    seen     map[uint64]bool
    dedupe   bool
    mu       sync.Mutex
}
```

Result processor features:
- Deduplication support
- Thread-safe processing
- Efficient result delivery

### File Operations

#### Operation Processor
```go
type ProcessorOptions struct {
    Workers          int
    MaxQueueSize     int
    ThrottleInterval time.Duration
}
```

File operation features:
- Concurrent processing
- Operation throttling
- Progress tracking
- Error handling

### Utility Functions

#### Size Parsing
```go
// Parse size strings (e.g., "1KB", "1.5MB", "2GB")
func ParseSize(s string) (int64, error)
```

#### Age Parsing
```go
// Parse age strings (e.g., "1h", "2d", "1w", "1m")
func ParseAge(s string) (time.Duration, error)
```

#### List Parsing
```go
// Split comma-separated strings
func SplitCommaList(s string) []string
```

### Current Limitations
- No content-based search
- Limited archive file support
- Basic file operation recovery
- Simple pattern matching (no regex)
- Limited network drive optimization

