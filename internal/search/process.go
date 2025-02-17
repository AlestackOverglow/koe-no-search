package search

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"runtime"
	"time"
	"sync/atomic"
)

// ProcessorOptions configures the file operation processor
type ProcessorOptions struct {
	Workers          int
	MaxQueueSize     int
	ThrottleInterval time.Duration
}

// processRegularFile processes a regular file
func processRegularFile(path string, info os.FileInfo, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor, buf []byte) {
	// Check if file exists and is not a symlink
	fi, err := os.Lstat(path)
	if err != nil {
		logDebug("File no longer exists or inaccessible: %s: %v", path, err)
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		logDebug("Skipping symbolic link: %s", path)
		return
	}

	if matchesPatterns(path, patterns, opts.IgnoreCase) &&
		matchesFileConstraints(info, opts) {
		
		var hash uint64
		var hashErr error
		
		// Safe hash calculation
		func() {
			defer func() {
				if r := recover(); r != nil {
					logError("Panic while calculating hash for %s: %v", path, r)
					hashErr = fmt.Errorf("hash calculation failed: %v", r)
				}
			}()
			hash = calculateQuickHash(path, info, buf)
			if hash == 0 {
				hashErr = fmt.Errorf("hash calculation returned zero value")
			}
		}()
		
		if hashErr != nil {
			logError("Failed to calculate hash for %s: %v", path, hashErr)
			return
		}
		
		result := SearchResult{
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    hash,
		}
		
		processor.add(result)
	}
}

// HandleFileOperation processes the file according to the specified operation
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

// checkFileAccess verifies if a file is accessible
func checkFileAccess(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if closeErr := f.Close(); closeErr != nil {
		return fmt.Errorf("error closing file: %v", closeErr)
	}
	return nil
}

// checkDirWritable verifies if a directory is writable
func checkDirWritable(dir string) error {
	// Generate shorter random name
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Errorf("failed to generate random name: %v", err)
	}
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp_%x", randomBytes))
	
	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	
	// Ensure cleanup in case of panic
	defer os.Remove(tmpFile)
	
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing test file: %v", err)
	}
	
	return os.Remove(tmpFile)
}

// checkFileWritable verifies if a file can be modified using platform-specific checks
func checkFileWritable(path string) error {
	// Try to open file with write permission
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("file is not writable: permission denied")
		}
		return err
	}
	f.Close()
	return nil
}

// Platform-specific constants for direct I/O
const (
	// Windows constants
	fileNoBuffering = 0x20000000 // FILE_FLAG_NO_BUFFERING

	// Default buffer size for aligned I/O
	directIOAlignment = 4096
	minBufferSize     = 4096
	defaultBufferSize = 128 * 1024  // 128KB default
	maxBufferSize     = 1024 * 1024 // 1MB max
	maxPoolSize       = 32          // Максимальное количество буферов в пуле
	
	// Platform-specific O_DIRECT values
	linuxODirect   = 0x4000   // Linux
	darwinODirect  = 0x100000 // Darwin/macOS
)

// BufferPool represents a pool of byte buffers
type BufferPool struct {
	pool    sync.Pool
	count   int32
	maxSize int32
}

// Global buffer pool instance
var copyBufferPool = &BufferPool{
	pool: sync.Pool{
		New: func() interface{} {
			return make([]byte, defaultBufferSize)
		},
	},
	maxSize: maxPoolSize,
}

func (p *BufferPool) Get() []byte {
	if atomic.LoadInt32(&p.count) < p.maxSize {
		if buf := p.pool.Get(); buf != nil {
			atomic.AddInt32(&p.count, 1)
			return buf.([]byte)
		}
	}
	return make([]byte, defaultBufferSize)
}

func (p *BufferPool) Put(buf []byte) {
	if buf == nil || len(buf) != defaultBufferSize {
		return
	}
	if atomic.LoadInt32(&p.count) < p.maxSize {
		p.pool.Put(buf)
	}
}

// optimizeBufferSize returns optimal buffer size for file operations
func optimizeBufferSize(fileSize int64) int {
	switch {
	case fileSize <= 0:
		return defaultBufferSize
	case fileSize < minBufferSize:
		return minBufferSize
	case fileSize < defaultBufferSize:
		return int(fileSize)
	default:
		return defaultBufferSize
	}
}

// enableDirectIO attempts to enable direct I/O on supported platforms
func enableDirectIO(file *os.File) {
	// Direct I/O is handled differently on each platform:
	// - Windows: Must be set when opening the file (FILE_FLAG_NO_BUFFERING)
	// - Linux/Darwin: Can be set after opening via fcntl
	// Since we can't modify flags after opening on Windows,
	// this function is effectively a no-op on Windows
	if runtime.GOOS == "windows" {
		return
	}

	// For non-Windows platforms, we'll use the file as is
	// Direct I/O will be handled through the open flags
	// This is a simplified approach that avoids platform-specific syscalls
	return
}

// getDirectIOFlags returns platform-specific direct I/O flags
func getDirectIOFlags() int {
	switch runtime.GOOS {
	case "windows":
		return fileNoBuffering
	case "linux":
		return linuxODirect
	case "darwin":
		return darwinODirect
	default:
		return 0
	}
}

// copyFile copies a file with optimized buffering
func copyFile(src string, opts FileOperationOptions, srcInfo os.FileInfo) error {
	if src == "" || srcInfo == nil {
		return fmt.Errorf("invalid arguments")
	}

	targetPath := filepath.Join(opts.TargetDir, filepath.Base(src))
	targetPath = resolveConflict(targetPath, opts.ConflictPolicy)
	if targetPath == "" {
		return nil
	}

	// Проверяем размер файла
	size := srcInfo.Size()
	if size == 0 {
		return copyEmptyFile(src, targetPath, srcInfo.Mode())
	}

	// Получаем буфер из пула
	buf := copyBufferPool.Get()
	defer copyBufferPool.Put(buf)

	// Открываем исходный файл
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open source: %v", err)
	}
	defer srcFile.Close()

	// Создаем временный файл
	tmpPath := targetPath + ".tmp"
	dstFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create target: %v", err)
	}

	success := false
	defer func() {
		if !success && tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	// Копируем данные
	written, err := io.CopyBuffer(dstFile, srcFile, buf)
	if err != nil {
		dstFile.Close()
		return fmt.Errorf("copy failed: %v", err)
	}

	if written != size {
		dstFile.Close()
		return fmt.Errorf("size mismatch: expected %d, got %d", size, written)
	}

	if err := dstFile.Sync(); err != nil {
		dstFile.Close()
		return fmt.Errorf("sync failed: %v", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("close failed: %v", err)
	}

	// Атомарное переименование
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("rename failed: %v", err)
	}

	success = true
	return nil
}

// Helper functions for copyFile
func copyEmptyFile(src, dst string, mode os.FileMode) error {
	if src == "" || dst == "" {
		return fmt.Errorf("source or destination path is empty")
	}
	
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create empty file: %v", err)
	}
	defer f.Close()
	return nil
}

func shouldUseDirectIO(fileSize int64) bool {
	return fileSize > 100*1024*1024 // 100MB
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

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file after copy: %v", err)
	}

	return nil
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
			timestamp := time.Now().UnixNano()
			
			// Try timestamp-based names first
			for i := 1; i <= 100; i++ {
				newPath := fmt.Sprintf("%s_%d_%d%s", base, timestamp, i, ext)
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					return newPath
				}
			}
			
			// If still no success, use random suffix
			randomBytes := make([]byte, 8)
			if _, err := rand.Read(randomBytes); err != nil {
				// If random generation fails, use timestamp as fallback
				return fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
			}
			return fmt.Sprintf("%s_%x%s", base, randomBytes, ext)
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
	started  bool
	stopped  bool
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	
	// Новые поля для контроля нагрузки
	maxQueueSize   int
	currentWorkers int32
	maxWorkers     int32
	throttle       *time.Ticker
}

type fileOperation struct {
	path string
	opts FileOperationOptions
	info os.FileInfo
}

// NewFileOperationProcessor creates a new processor with specified options
func NewFileOperationProcessor(opts ProcessorOptions) *FileOperationProcessor {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}
	if opts.MaxQueueSize <= 0 {
		opts.MaxQueueSize = 1000
	}
	if opts.ThrottleInterval <= 0 {
		opts.ThrottleInterval = 100 * time.Millisecond
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	return &FileOperationProcessor{
		opChan:         make(chan fileOperation, opts.MaxQueueSize),
		workers:        opts.Workers,
		maxWorkers:     int32(opts.Workers * 2), // Позволяем удвоить количество при необходимости
		stopChan:       make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
		throttle:      time.NewTicker(opts.ThrottleInterval),
		maxQueueSize:  opts.MaxQueueSize,
	}
}

// Start begins processing file operations
func (p *FileOperationProcessor) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.started {
		return fmt.Errorf("processor already started")
	}
	if p.stopped {
		return fmt.Errorf("processor has been stopped")
	}
	
	p.started = true
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	
	return nil
}

// Stop gracefully stops all workers
func (p *FileOperationProcessor) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.cancel()
	close(p.stopChan)
	p.mu.Unlock()
	
	p.wg.Wait()
	
	p.mu.Lock()
	if p.opChan != nil {
		close(p.opChan)
		p.opChan = nil
	}
	p.mu.Unlock()
}

// Add queues a file operation for processing with backpressure
func (p *FileOperationProcessor) Add(path string, opts FileOperationOptions, info os.FileInfo) error {
	if p == nil {
		return fmt.Errorf("processor is nil")
	}
	
	if path == "" || info == nil {
		return fmt.Errorf("invalid arguments: path or file info is nil")
	}

	p.mu.Lock()
	if p.stopped || p.opChan == nil {
		p.mu.Unlock()
		return fmt.Errorf("processor is stopped")
	}
	p.mu.Unlock()

	// Применяем throttling
	<-p.throttle.C

	select {
	case p.opChan <- fileOperation{path, opts, info}:
		logDebug("Queued file operation for %s", path)
		return nil
	case <-p.stopChan:
		return fmt.Errorf("processor is stopping")
	case <-p.ctx.Done():
		return fmt.Errorf("processor context cancelled")
	default:
		// Пытаемся адаптивно увеличить количество воркеров
		if atomic.LoadInt32(&p.currentWorkers) < p.maxWorkers {
			p.mu.Lock()
			if !p.stopped {
				p.wg.Add(1)
				atomic.AddInt32(&p.currentWorkers, 1)
				go p.worker()
			}
			p.mu.Unlock()
		}
		return fmt.Errorf("operation queue is full")
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
			// Create timeout context for each operation
			ctx, cancel := context.WithTimeout(p.ctx, 30*time.Minute)
			err := p.processOperation(ctx, op)
			cancel()
			if err != nil {
				logError("Failed to process file operation for %s: %v", op.path, err)
			}
		case <-p.stopChan:
			return
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *FileOperationProcessor) processOperation(ctx context.Context, op fileOperation) error {
	done := make(chan error, 1)
	go func() {
		done <- HandleFileOperation(op.path, op.opts)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out or cancelled")
	}
}

// checkDiskSpace verifies if there's enough space for the file with timeout
func checkDiskSpace(path string, size int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		dir := filepath.Dir(path)
		tmpPath := filepath.Join(dir, fmt.Sprintf(".space_check_%d", time.Now().UnixNano()))
		
		f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			done <- fmt.Errorf("failed to create test file: %v", err)
			return
		}
		defer func() {
			f.Close()
			os.Remove(tmpPath)
		}()

		// Проверяем минимальный размер или 1MB, что меньше
		checkSize := int64(1024 * 1024)
		if size < checkSize {
			checkSize = size
		}

		if err := f.Truncate(checkSize); err != nil {
			done <- fmt.Errorf("insufficient disk space: %v", err)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("disk space check timed out")
	}
} 