# Koe no Search - API Documentation

This document describes the public API of the Koe no Search library, which you can use to integrate file search capabilities into your Go applications.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Types](#core-types)
- [Search Functions](#search-functions)
- [File Operations](#file-operations)
- [Utilities](#utilities)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Benchmarks](#benchmarks)
- [Migration Guide](#migration-guide)
- [Troubleshooting](#troubleshooting)

## Installation

```bash
go get github.com/yourusername/koe-no-search
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
    "filesearch/internal/search"
)

func main() {
    opts := search.SearchOptions{
        RootDirs:   []string{"/path/to/search"},
        Patterns:   []string{"*.txt", "*.doc"},
        Extensions: []string{"txt", "doc"},
        MaxWorkers: 4,
        IgnoreCase: true,
    }

    results := search.Search(opts)
    for result := range results {
        if result.Error != nil {
            fmt.Printf("Error: %v\n", result.Error)
            continue
        }
        fmt.Printf("Found: %s (Size: %d bytes)\n", result.Path, result.Size)
    }
}
```

## Core Types

### SearchOptions

```go
type SearchOptions struct {
    RootDirs         []string        // List of root directories to search
    Patterns         []string        // List of search patterns (e.g., "*.txt")
    Extensions       []string        // List of file extensions without dot
    MaxWorkers       int            // Number of concurrent workers
    IgnoreCase       bool           // Case-insensitive search
    BufferSize       int            // Channel buffer size
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
    StopChan         chan struct{}  // Channel for stopping the search
    FileOp           FileOperationOptions
    ExcludeDirs      []string       // Directories to exclude from search
}
```

#### Default Values
- `MaxWorkers`: Number of CPU cores
- `BufferSize`: 1000
- `BatchSize`: 100
- `MinMMapSize`: 1MB (1024*1024 bytes)

### SearchResult

```go
type SearchResult struct {
    Path      string      // Full path to the file
    Size      int64       // File size in bytes
    Mode      os.FileMode // File mode and permissions
    ModTime   time.Time   // Last modification time
    Hash      uint64      // Quick hash for duplicate detection
    Error     error       // Error if occurred during processing
}
```

### FileOperationOptions

```go
type FileOperationOptions struct {
    Operation       FileOperation
    TargetDir      string
    ConflictPolicy ConflictResolutionPolicy
}

type FileOperation int

const (
    NoOperation FileOperation = iota
    CopyFiles
    MoveFiles
    DeleteFiles
)

type ConflictResolutionPolicy int

const (
    Skip ConflictResolutionPolicy = iota
    Overwrite
    Rename
)
```

## Search Functions

### Search

```go
func Search(opts SearchOptions) chan SearchResult
```

The main search function that initiates the file search process. Returns a channel of search results.

**Features:**
- Concurrent file processing
- Memory-mapped file handling for large files
- Automatic worker management
- Progress tracking
- Graceful cancellation support

**Example:**
```go
opts := search.SearchOptions{
    RootDirs:   []string{"/home/user"},
    Patterns:   []string{"*.go"},
    MaxWorkers: 4,
}

results := search.Search(opts)
for result := range results {
    // Process results
}
```

### Advanced Search Example

```go
opts := search.SearchOptions{
    RootDirs:         []string{"/data"},
    Patterns:         []string{"*.log"},
    Extensions:       []string{"log"},
    MaxWorkers:       runtime.NumCPU(),
    IgnoreCase:       true,
    MinSize:          1024 * 1024, // 1MB
    MaxAge:           24 * time.Hour,
    ExcludeHidden:    true,
    DeduplicateFiles: true,
    UseMMap:          true,
    MinMMapSize:      10 * 1024 * 1024, // 10MB
    ExcludeDirs:      []string{"/data/temp"},
}

stopChan := make(chan struct{})
opts.StopChan = stopChan

go func() {
    time.Sleep(30 * time.Second)
    close(stopChan) // Stop search after 30 seconds
}()

results := search.Search(opts)
for result := range results {
    // Process results
}
```

## File Operations

### HandleFileOperation

```go
func HandleFileOperation(path string, opts FileOperationOptions) error
```

Processes a single file according to the specified operation (copy, move, or delete).

**Features:**
- Atomic operations where possible
- Automatic directory creation
- Conflict resolution
- Permission preservation
- Error recovery

**Example:**
```go
opts := search.FileOperationOptions{
    Operation:       search.CopyFiles,
    TargetDir:      "/path/to/target",
    ConflictPolicy: search.Rename,
}

err := search.HandleFileOperation("/path/to/file.txt", opts)
if err != nil {
    log.Printf("Error: %v", err)
}
```

### FileOperationProcessor

```go
type FileOperationProcessor struct {
    // ... internal fields
}

func NewFileOperationProcessor(workers int) *FileOperationProcessor
func (p *FileOperationProcessor) Start()
func (p *FileOperationProcessor) Stop()
func (p *FileOperationProcessor) Add(path string, opts FileOperationOptions, info os.FileInfo)
```

Handles file operations asynchronously with multiple workers.

**Features:**
- Concurrent processing
- Automatic queue management
- Graceful shutdown
- Error handling per file

**Example:**
```go
processor := search.NewFileOperationProcessor(4)
processor.Start()
defer processor.Stop()

opts := search.FileOperationOptions{
    Operation:       search.CopyFiles,
    TargetDir:      "/backup",
    ConflictPolicy: search.Rename,
}

processor.Add("/path/to/file.txt", opts, fileInfo)
```

## Utilities

### Logger

```go
func InitLogger()
func CloseLogger()
func LogDebug(format string, args ...interface{})
func LogInfo(format string, args ...interface{})
func LogWarning(format string, args ...interface{})
func LogError(format string, args ...interface{})
```

Logging utilities for debugging and error tracking.

**Features:**
- Log levels (DEBUG, INFO, WARNING, ERROR)
- File-based logging
- Timestamp and source file information
- Thread-safe logging

**Example:**
```go
search.InitLogger()
defer search.CloseLogger()

search.LogInfo("Starting search in %s", searchDir)
```

## Advanced Features

### Memory Management

The library implements several memory optimization techniques:

1. **Buffer Pooling**
   ```go
   opts.BufferSize = 32 * 1024 // 32KB buffers
   ```

2. **Memory Mapping**
   ```go
   opts.UseMMap = true
   opts.MinMMapSize = 50 * 1024 * 1024 // 50MB threshold
   ```

3. **Batch Processing**
   ```go
   opts.BatchSize = 1000 // Process files in batches of 1000
   ```

### Duplicate Detection

The library uses a fast hashing algorithm for duplicate detection:

```go
opts.DeduplicateFiles = true
```

The hash is calculated using:
- File size
- First 1KB of content
- Modification time
- Path information

### Pattern Matching

Supports various pattern matching methods:

1. **Simple patterns**
   ```go
   opts.Patterns = []string{"*.txt", "doc_*.pdf"}
   ```

2. **Extensions**
   ```go
   opts.Extensions = []string{"jpg", "png", "gif"}
   ```

3. **Case sensitivity**
   ```go
   opts.IgnoreCase = true
   ```

## Best Practices

### Resource Management

1. **Worker Count**
   ```go
   opts.MaxWorkers = runtime.NumCPU() // For CPU-bound tasks
   opts.MaxWorkers = runtime.NumCPU() * 2 // For I/O-bound tasks
   ```

2. **Buffer Sizes**
   ```go
   opts.BufferSize = 1000 // Default
   opts.BufferSize = 10000 // For high-throughput searches
   ```

3. **Memory Mapping**
   ```go
   opts.UseMMap = true
   opts.MinMMapSize = 1024 * 1024 // 1MB threshold
   ```

### Error Handling

1. **Graceful Shutdown**
   ```go
   stopChan := make(chan struct{})
   opts.StopChan = stopChan
   // ... later
   close(stopChan) // Stop search gracefully
   ```

2. **Error Processing**
   ```go
   for result := range results {
       if result.Error != nil {
           switch {
           case os.IsPermission(result.Error):
               // Handle permission error
           case os.IsNotExist(result.Error):
               // Handle missing file
           default:
               // Handle other errors
           }
           continue
       }
       // Process valid result
   }
   ```

## Benchmarks

Performance measurements on typical systems:

| Operation | Small Files (<1MB) | Large Files (>1GB) |
|-----------|-------------------|-------------------|
| Search    | ~10,000 files/s   | ~100 files/s     |
| Copy      | ~1,000 files/s    | ~10 files/s      |
| Move      | ~2,000 files/s    | ~20 files/s      |
| Delete    | ~5,000 files/s    | ~50 files/s      |

*Tested on Windows 10, Intel i7, 16GB RAM, SSD

## Migration Guide

### Upgrading from v1.x to v2.x

1. **SearchOptions Changes**
   - Added `ExcludeDirs`
   - Renamed `MaxThreads` to `MaxWorkers`

2. **File Operations**
   - New `FileOperationProcessor`
   - Enhanced conflict resolution

3. **Logger**
   - New logging interface
   - Added log levels

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   ```go
   // Reduce memory usage
   opts.BatchSize = 50
   opts.BufferSize = 500
   opts.UseMMap = false
   ```

2. **Slow Performance**
   ```go
   // Optimize for speed
   opts.MaxWorkers = runtime.NumCPU() * 2
   opts.UseMMap = true
   opts.MinMMapSize = 1024 * 1024
   ```

3. **Missing Files**
   ```go
   // Ensure all files are found
   opts.FollowSymlinks = true
   opts.ExcludeHidden = false
   ```

### Debugging

Enable detailed logging:
```go
search.InitLogger()
search.LogDebug("Starting search with options: %+v", opts)
```

## Support

For issues and feature requests, please visit:
https://github.com/yourusername/koe-no-search/issues 