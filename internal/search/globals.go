package search

import (
	"sync"
)

var (
	globalCache = newCache()
	
	// Common binary and temporary file extensions to skip
	skipExtensions = map[string]bool{}
	
	// Common directories to skip
	skipDirs = map[string]bool{
		"node_modules": true, ".git": true, ".svn": true,
		"target": true, "build": true, "dist": true,
		"__pycache__": true, ".idea": true, ".vscode": true,
		"$RECYCLE.BIN": true, "System Volume Information": true,
		"Windows": true, "Program Files": true, "Program Files (x86)": true,
		"ProgramData": true, "AppData": true, "Recovery": true,
	}
	
	// File read buffer pool
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}
	
	fileIndex = &FileIndex{
		Files:    make(map[string]*FileMetadata),
		DirStats: make(map[string]*DirStats),
	}
	
	// Pool for processing large files
	mmapPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024*1024) // 1MB buffer
		},
	}
	
	// Priority queues for files
	highPriorityPaths = make(chan string, 10000)
	normalPriorityPaths = make(chan string, 10000)
	lowPriorityPaths = make(chan string, 10000)
) 