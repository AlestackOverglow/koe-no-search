package search

import (
	"sync"
)

// Version information
var (
	Version = "0.2.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
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
		"Documents and Settings": true, // Legacy Windows directory
		"System32": true,
		"SysWOW64": true,
		"WindowsApps": true,
		"WinSxS": true,
	}
	
	// Global file filter set
	globalFileFilter = NewFileFilterSet()
	
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