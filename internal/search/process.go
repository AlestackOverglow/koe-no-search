package search

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"runtime"
)

// processRegularFile processes a regular file
func processRegularFile(path string, info os.FileInfo, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor, buf []byte) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		logDebug("File no longer exists or inaccessible: %s: %v", path, err)
		return
	}

	if matchesPatterns(path, patterns, opts.IgnoreCase) &&
		matchesFileConstraints(info, opts) {
		
		var hash uint64
		var err error
		
		// Safe hash calculation
		func() {
			defer func() {
				if r := recover(); r != nil {
					logError("Panic while calculating hash for %s: %v", path, r)
					err = fmt.Errorf("hash calculation failed: %v", r)
				}
			}()
			hash = calculateQuickHash(path, info, buf)
		}()
		
		result := SearchResult{
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    hash,
			Error:   err,
		}
		
		processor.add(result)
	}
}

// handleFileOperation processes the file according to the specified operation
func handleFileOperation(path string, opts FileOperationOptions) error {
	if opts.Operation == NoOperation {
		return nil
	}

	// Get file info for source
	srcInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %v", err)
	}

	// Check if file is accessible
	if err := checkFileAccess(path); err != nil {
		return fmt.Errorf("file is not accessible: %v", err)
	}

	// Create target directory if it doesn't exist
	if opts.Operation != DeleteFiles {
		if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %v", err)
		}

		// Check if target directory is writable
		if err := checkDirWritable(opts.TargetDir); err != nil {
			return fmt.Errorf("target directory is not writable: %v", err)
		}
	}

	switch opts.Operation {
	case CopyFiles:
		return copyFile(path, opts, srcInfo)
	case MoveFiles:
		return moveFile(path, opts, srcInfo)
	case DeleteFiles:
		// Check if file is writable before attempting to delete
		if err := checkFileWritable(path); err != nil {
			return fmt.Errorf("file is not writable: %v", err)
		}
		return os.Remove(path)
	default:
		return fmt.Errorf("unknown operation: %v", opts.Operation)
	}
}

// checkFileAccess verifies if a file is accessible
func checkFileAccess(path string) error {
	// Try to open file for reading
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// checkFileWritable verifies if a file can be modified
func checkFileWritable(path string) error {
	// Check if file exists and is writable
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Check if file is read-only
	if info.Mode()&0200 == 0 {
		return fmt.Errorf("file is read-only")
	}

	return nil
}

// checkDirWritable verifies if a directory is writable
func checkDirWritable(dir string) error {
	// Create a temporary file to test write permissions
	tmpFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	f.Close()
	
	// Clean up test file
	return os.Remove(tmpFile)
}

// copyFile copies a file to the target directory
func copyFile(src string, opts FileOperationOptions, srcInfo os.FileInfo) error {
	targetPath := filepath.Join(opts.TargetDir, filepath.Base(src))
	targetPath = resolveConflict(targetPath, opts.ConflictPolicy)
	if targetPath == "" {
		return nil // Skip if conflict resolution returned empty path
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	// Create target file
	dstFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer dstFile.Close()

	// Use buffered copy for better performance
	buf := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	return nil
}

// moveFile moves a file to the target directory
func moveFile(src string, opts FileOperationOptions, srcInfo os.FileInfo) error {
	targetPath := filepath.Join(opts.TargetDir, filepath.Base(src))
	targetPath = resolveConflict(targetPath, opts.ConflictPolicy)
	if targetPath == "" {
		return nil // Skip if conflict resolution returned empty path
	}

	// Check if source file is writable before attempting to move
	if err := checkFileWritable(src); err != nil {
		return fmt.Errorf("source file is not writable: %v", err)
	}

	// Try to move the file directly first
	err := os.Rename(src, targetPath)
	if err == nil {
		return nil
	}

	// If direct move fails, try copy and delete
	if err := copyFile(src, opts, srcInfo); err != nil {
		return fmt.Errorf("failed to copy file during move: %v", err)
	}

	return os.Remove(src)
}

// resolveConflict handles file name conflicts according to the policy
func resolveConflict(path string, policy ConflictResolutionPolicy) string {
	if policy == Overwrite {
		return path
	}

	if _, err := os.Stat(path); err == nil {
		switch policy {
		case Skip:
			return ""
		case Rename:
			ext := filepath.Ext(path)
			base := strings.TrimSuffix(path, ext)
			counter := 1
			for {
				newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
				if _, err := os.Stat(newPath); err != nil {
					return newPath
				}
				counter++
			}
		}
	}

	return path
}

// FileOperationProcessor handles file operations asynchronously
type FileOperationProcessor struct {
	opChan   chan fileOperation
	workers  int
	wg       sync.WaitGroup
	stopChan chan struct{}
}

type fileOperation struct {
	path string
	opts FileOperationOptions
	info os.FileInfo
}

// NewFileOperationProcessor creates a new processor with specified number of workers
func NewFileOperationProcessor(workers int) *FileOperationProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	
	return &FileOperationProcessor{
		opChan:   make(chan fileOperation, 1000),
		workers:  workers,
		stopChan: make(chan struct{}),
	}
}

// Start begins processing file operations
func (p *FileOperationProcessor) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Stop gracefully stops all workers
func (p *FileOperationProcessor) Stop() {
	close(p.stopChan)
	close(p.opChan)
	p.wg.Wait()
}

// HandleFileOperation processes a single file according to the specified operation
func HandleFileOperation(path string, opts FileOperationOptions) error {
	if opts.Operation == NoOperation {
		return nil
	}

	// Get file info for source
	srcInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %v", err)
	}

	// Check if file is accessible
	if err := checkFileAccess(path); err != nil {
		return fmt.Errorf("file is not accessible: %v", err)
	}

	// Create target directory if it doesn't exist
	if opts.Operation != DeleteFiles {
		if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %v", err)
		}

		// Check if target directory is writable
		if err := checkDirWritable(opts.TargetDir); err != nil {
			return fmt.Errorf("target directory is not writable: %v", err)
		}
	}

	switch opts.Operation {
	case CopyFiles:
		return copyFile(path, opts, srcInfo)
	case MoveFiles:
		return moveFile(path, opts, srcInfo)
	case DeleteFiles:
		// Check if file is writable before attempting to delete
		if err := checkFileWritable(path); err != nil {
			return fmt.Errorf("file is not writable: %v", err)
		}
		return os.Remove(path)
	default:
		return fmt.Errorf("unknown operation: %v", opts.Operation)
	}
}

// worker processes file operations from the queue
func (p *FileOperationProcessor) worker() {
	defer p.wg.Done()
	
	for {
		select {
		case op, ok := <-p.opChan:
			if !ok {
				return
			}
			if err := HandleFileOperation(op.path, op.opts); err != nil {
				logError("Failed to process file operation for %s: %v", op.path, err)
			}
		case <-p.stopChan:
			return
		}
	}
}

// Add queues a file operation for processing
func (p *FileOperationProcessor) Add(path string, opts FileOperationOptions, info os.FileInfo) {
	select {
	case p.opChan <- fileOperation{path, opts, info}:
		logDebug("Queued file operation for %s", path)
	case <-p.stopChan:
		logDebug("File operation processor stopped, skipping %s", path)
	default:
		logWarning("File operation queue full, skipping %s", path)
	}
} 