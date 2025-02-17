# Koe no Search - API Documentation

This document describes the public API of the Koe no Search library, which you can use to integrate file search capabilities into your Go applications.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Types](#core-types)
- [Search Functions](#search-functions)
- [File Operations](#file-operations)
- [Error Handling](#error-handling)
- [Performance Optimization](#performance-optimization)
- [Memory Management](#memory-management)
- [Security Considerations](#security-considerations)
- [Testing](#testing)
- [Best Practices](#best-practices)
- [Benchmarks](#benchmarks)
- [Migration Guide](#migration-guide)
- [Troubleshooting](#troubleshooting)
- [Platform-Specific Notes](#platform-specific-notes)

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
    "context"
    "time"
    "filesearch/internal/search"
)

func main() {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Create stop channel for graceful cancellation
    stopChan := make(chan struct{})
    go func() {
        <-ctx.Done()
        close(stopChan)
    }()

    opts := search.SearchOptions{
        RootDirs:   []string{"/path/to/search"},
        Patterns:   []string{"*.txt", "*.doc"},
        Extensions: []string{"txt", "doc"},
        MaxWorkers: 4,
        IgnoreCase: true,
        StopChan:   stopChan,
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

#### Default Values and Behavior
- `MaxWorkers`: Number of CPU cores (runtime.NumCPU())
- `BufferSize`: 1000
- `BatchSize`: 100
- `MinMMapSize`: 1MB (1024*1024 bytes)
- `FollowSymlinks`: false (for security reasons)
- `UseMMap`: false (enabled manually for performance)
- `DeduplicateFiles`: false (optional memory-intensive feature)

#### Performance Impact of Options
- `MaxWorkers`: More workers can improve performance on SSDs but may degrade performance on HDDs
- `BufferSize`: Larger buffers consume more memory but can improve performance for large directories
- `BatchSize`: Larger batches reduce overhead but increase memory usage
- `UseMMap`: Improves performance for large files but increases memory usage
- `DeduplicateFiles`: Significant memory overhead, use carefully with large file sets

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

## Error Handling

### Common Error Types
```go
var (
    ErrAccessDenied     = errors.New("access denied")
    ErrNotFound         = errors.New("file not found")
    ErrInvalidPattern   = errors.New("invalid search pattern")
    ErrCancelled        = errors.New("search cancelled")
    ErrMemoryLimit      = errors.New("memory limit exceeded")
)
```

### Error Handling Examples
```go
results := search.Search(opts)
for result := range results {
    switch {
    case errors.Is(result.Error, ErrAccessDenied):
        // Handle permission issues
        log.Printf("Permission denied: %s", result.Path)
    case errors.Is(result.Error, ErrNotFound):
        // Handle missing files
        log.Printf("File not found: %s", result.Path)
    case result.Error != nil:
        // Handle other errors
        log.Printf("Error processing %s: %v", result.Path, result.Error)
    default:
        // Process successful result
        processFile(result)
    }
}
```

## Performance Optimization

### Memory Mapping
Memory mapping is beneficial for large files (>1MB) as it:
- Reduces memory usage
- Improves search speed
- Allows partial file reading

Example with memory mapping:
```go
opts := search.SearchOptions{
    RootDirs:    []string{"/data"},
    UseMMap:     true,
    MinMMapSize: 1024 * 1024, // 1MB
}
```

### Worker Configuration
Optimal worker count depends on:
- CPU cores available
- Disk type (SSD/HDD)
- File system type
- Available memory

Guidelines:
- SSDs: Use runtime.NumCPU() * 2
- HDDs: Use runtime.NumCPU()
- Network drives: Use runtime.NumCPU() / 2

### Batch Processing
Batch processing reduces system call overhead:
```go
opts := search.SearchOptions{
    BatchSize: 1000, // Increased batch size for better performance
}
```

## Memory Management

### Memory Usage Patterns
- Search process: O(n) where n is batch size
- Deduplication: O(m) where m is number of files
- Memory mapping: Virtual memory, not RAM

### Memory Optimization
```go
// Optimize for low memory usage
opts := search.SearchOptions{
    BatchSize:        50,    // Smaller batches
    DeduplicateFiles: false, // Disable deduplication
    UseMMap:         true,   // Use memory mapping
}

// Optimize for speed
opts := search.SearchOptions{
    BatchSize:        1000,  // Larger batches
    DeduplicateFiles: true,  // Enable deduplication
    BufferSize:       5000,  // Larger buffers
}
```

## Security Considerations

### Symbolic Links
By default, symbolic links are not followed for security reasons. To enable:
```go
opts := search.SearchOptions{
    FollowSymlinks: true, // Enable with caution
}
```

### File Permissions
The library respects file system permissions:
- Skips files/directories without read permission
- Reports permission errors in SearchResult.Error
- Preserves file permissions in copy operations

### Safe File Operations
```go
// Safe file copy with permission preservation
opts := search.FileOperationOptions{
    Operation:       search.CopyFiles,
    ConflictPolicy: search.Skip, // Don't overwrite existing files
}
```

## Testing

### Unit Testing
```go
func TestSearch(t *testing.T) {
    opts := search.SearchOptions{
        RootDirs: []string{"testdata"},
        Patterns: []string{"*.txt"},
    }
    
    results := search.Search(opts)
    var found []string
    for result := range results {
        if result.Error != nil {
            t.Errorf("Search error: %v", result.Error)
            continue
        }
        found = append(found, result.Path)
    }
    
    expected := []string{"testdata/file1.txt", "testdata/file2.txt"}
    if !reflect.DeepEqual(found, expected) {
        t.Errorf("Expected %v, got %v", expected, found)
    }
}
```

### Mock File System
```go
type MockFS struct {
    files map[string][]byte
}

func (m *MockFS) Open(name string) (io.ReadCloser, error) {
    data, ok := m.files[name]
    if !ok {
        return nil, os.ErrNotExist
    }
    return io.NopCloser(bytes.NewReader(data)), nil
}
```

## Best Practices

### Resource Management
1. Always use StopChan for cancellation
2. Close channels properly
3. Use context for timeout management
4. Release memory-mapped files

### Error Handling
1. Check SearchResult.Error for each result
2. Use appropriate conflict policies
3. Implement proper logging
4. Handle system-specific errors

### Performance
1. Adjust worker count based on system
2. Use memory mapping for large files
3. Implement proper batching
4. Monitor memory usage

### Security
1. Validate file paths
2. Check file permissions
3. Handle symbolic links safely
4. Implement proper access controls

## Benchmarks

### System Specifications
- CPU: Intel i7 (8 cores)
- RAM: 16GB
- Storage: NVMe SSD
- OS: Linux 5.15

### Results
| Scenario | Files | Time | Memory |
|----------|-------|------|---------|
| Small files | 10,000 | 0.5s | 50MB |
| Large files | 1,000 | 2.0s | 200MB |
| Mixed sizes | 50,000 | 5.0s | 150MB |

### Configuration Impact
| Setting | Performance Impact | Memory Impact |
|---------|-------------------|---------------|
| Workers +50% | +20% speed | +10% memory |
| Batch x2 | +15% speed | +25% memory |
| MMap | +30% speed | -20% memory |

## Platform-Specific Notes

### Windows
- Uses Win32 API for file operations
- Handles drive letters and UNC paths
- Special handling for hidden files
- NTFS symbolic link support

### Linux
- Uses inotify for file system events
- Handles file system mount points
- Supports extended attributes
- Handles various file systems

### macOS
- Uses FSEvents for file system monitoring
- Handles resource forks
- Supports Time Machine volumes
- Special handling for packages

## Troubleshooting

### Common Issues

1. Performance Problems
```go
// Solution: Adjust worker count and batch size
opts := search.SearchOptions{
    MaxWorkers: runtime.NumCPU() / 2,
    BatchSize:  500,
}
```

2. Memory Issues
```go
// Solution: Reduce memory usage
opts := search.SearchOptions{
    BatchSize:        50,
    DeduplicateFiles: false,
    UseMMap:         true,
}
```

3. Permission Errors
```go
// Solution: Implement proper error handling
if errors.Is(err, ErrAccessDenied) {
    log.Printf("Permission denied: %s", path)
    // Skip or handle accordingly
}
```

### Debugging
```go
// Enable debug logging
search.SetLogLevel(search.LogLevelDebug)

// Add custom logger
search.SetLogger(func(level LogLevel, format string, args ...interface{}) {
    log.Printf(format, args...)
})
```

