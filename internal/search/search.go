package search

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// Search performs concurrent file search based on given options
func Search(opts SearchOptions) chan SearchResult {
	if opts.BufferSize <= 0 {
		opts.BufferSize = 1000
	}
	
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = runtime.NumCPU()
	}
	
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}

	// Clear cache before each search
	globalCache = newCache()
	fileIndex = &FileIndex{
		Files:    make(map[string]*FileMetadata),
		DirStats: make(map[string]*DirStats),
	}

	patterns := preparePatterns(opts)
	
	results := make(chan SearchResult, opts.BufferSize)
	paths := make(chan string, opts.BufferSize)
	
	// Create result processor
	processor := newResultProcessor(results, opts)
	
	// Create file operation processor if needed
	var fileOpProcessor *FileOperationProcessor
	if opts.FileOp.Operation != NoOperation {
		fileOpProcessor = NewFileOperationProcessor(ProcessorOptions{
			Workers:          opts.MaxWorkers / 2,
			MaxQueueSize:     1000,
			ThrottleInterval: 100 * time.Millisecond,
		})
		fileOpProcessor.Start()
	}
	
	var wg sync.WaitGroup
	
	// Start file matcher goroutines
	for i := 0; i < opts.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Create batch processor
			batchProc := newBatchProcessor(opts.BatchSize, func(batch []string) {
				processFileBatch(batch, patterns, opts, processor, fileOpProcessor)
			})
			
			// Process files
			for {
				select {
				case path, ok := <-paths:
					if !ok {
						batchProc.flush()
						return
					}
					if shouldProcessFile(path, opts) {
						batchProc.add(path)
					}
				case <-opts.StopChan:
					batchProc.flush()
					return
				}
			}
		}()
	}
	
	// Start directory walkers
	var walkWg sync.WaitGroup
	walkWg.Add(len(opts.RootDirs))
	
	for _, rootDir := range opts.RootDirs {
		go func(dir string) {
			defer walkWg.Done()
			walkDirectoryOptimized(dir, paths, opts)
		}(rootDir)
	}
	
	// Close channels after search completion
	go func() {
		walkWg.Wait()
		close(paths)
		wg.Wait()
		
		// Stop file operation processor if it was used
		if fileOpProcessor != nil {
			fileOpProcessor.Stop()
		}
		
		processor.close()
	}()
	
	// Enable aggressive GC mode
	debug.SetGCPercent(10)
	defer debug.SetGCPercent(100)
	
	return results
}

// processFileBatch processes a batch of files
func processFileBatch(batch []string, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor, fileOpProcessor *FileOperationProcessor) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	
	for _, path := range batch {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		// Check if file matches patterns and constraints before processing
		if !matchesPatterns(path, patterns, opts.IgnoreCase) ||
			!matchesFileConstraints(info, opts) {
			continue
		}
		
		// Use mmap for large files
		if opts.UseMMap && info.Size() >= opts.MinMMapSize {
			if err := processByMMap(path, info, patterns, opts, processor); err == nil {
				// Queue file operation if needed
				if fileOpProcessor != nil && opts.FileOp.Operation != NoOperation {
					fileOpProcessor.Add(path, opts.FileOp, info)
				}
				continue
			}
		}
		
		// Regular processing for other files
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
		}()
		
		result := SearchResult{
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    hash,
			Error:   hashErr,
		}
		
		processor.add(result)
		
		// Queue file operation if needed
		if fileOpProcessor != nil && opts.FileOp.Operation != NoOperation {
			fileOpProcessor.Add(path, opts.FileOp, info)
		}
	}
} 