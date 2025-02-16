package search

import (
	"os"
	"runtime"
	"runtime/debug"
	"sync"
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
	
	var wg sync.WaitGroup
	
	// Start file matcher goroutines
	for i := 0; i < opts.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Create batch processor
			batchProc := newBatchProcessor(opts.BatchSize, func(batch []string) {
				processFileBatch(batch, patterns, opts, processor)
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
		processor.close()
	}()
	
	// Enable aggressive GC mode
	debug.SetGCPercent(10)
	defer debug.SetGCPercent(100)
	
	return results
}

// processFileBatch processes a batch of files
func processFileBatch(batch []string, patterns compiledPatterns, opts SearchOptions, processor *resultProcessor) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	
	for _, path := range batch {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		
		// Use mmap for large files
		if opts.UseMMap && info.Size() >= opts.MinMMapSize {
			if err := processByMMap(path, info, patterns, opts, processor); err == nil {
				continue
			}
		}
		
		// Regular processing for other files
		processRegularFile(path, info, patterns, opts, processor, buf)
	}
} 