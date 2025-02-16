package search

import (
	"os"
)

// setupPriorityQueues sets up priority queues
func setupPriorityQueues(opts SearchOptions) {
	// Create priority map for quick lookup
	priorityMap := make(map[string]int)
	for _, dir := range opts.PriorityDirs {
		priorityMap[dir] = 2 // High priority
	}
	for _, dir := range opts.LowPriorityDirs {
		priorityMap[dir] = 0 // Low priority
	}
}

// startPriorityWorkers starts workers with different priorities
func startPriorityWorkers(opts SearchOptions, processor *resultProcessor) {
	// High priority
	for i := 0; i < opts.MaxWorkers/4; i++ {
		go processHighPriorityFiles(opts, processor)
	}
	
	// Normal priority
	for i := 0; i < opts.MaxWorkers/2; i++ {
		go processNormalPriorityFiles(opts, processor)
	}
	
	// Low priority
	for i := 0; i < opts.MaxWorkers/4; i++ {
		go processLowPriorityFiles(opts, processor)
	}
}

// processHighPriorityFiles processes high priority files
func processHighPriorityFiles(opts SearchOptions, processor *resultProcessor) {
	patterns := preparePatterns(opts)
	for path := range highPriorityPaths {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		
		if matchesPatterns(path, patterns, opts.IgnoreCase) &&
			matchesFileConstraints(info, opts) {
			
			hash := calculateQuickHash(path, info, bufferPool.Get().([]byte))
			
			processor.add(SearchResult{
				Path:    path,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				Hash:    hash,
			})
		}
	}
}

// processNormalPriorityFiles processes normal priority files
func processNormalPriorityFiles(opts SearchOptions, processor *resultProcessor) {
	patterns := preparePatterns(opts)
	for path := range normalPriorityPaths {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		
		if matchesPatterns(path, patterns, opts.IgnoreCase) &&
			matchesFileConstraints(info, opts) {
			
			hash := calculateQuickHash(path, info, bufferPool.Get().([]byte))
			
			processor.add(SearchResult{
				Path:    path,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				Hash:    hash,
			})
		}
	}
}

// processLowPriorityFiles processes low priority files
func processLowPriorityFiles(opts SearchOptions, processor *resultProcessor) {
	patterns := preparePatterns(opts)
	for path := range lowPriorityPaths {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		
		if matchesPatterns(path, patterns, opts.IgnoreCase) &&
			matchesFileConstraints(info, opts) {
			
			hash := calculateQuickHash(path, info, bufferPool.Get().([]byte))
			
			processor.add(SearchResult{
				Path:    path,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				Hash:    hash,
			})
		}
	}
} 