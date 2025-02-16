package search

import (
	"sync"
	"os"
	"encoding/binary"
	"github.com/cespare/xxhash"
)

// resultProcessor handles result processing and deduplication
type resultProcessor struct {
	results  chan<- SearchResult
	seen     map[uint64]bool
	dedupe   bool
	mu       sync.Mutex
}

func newResultProcessor(results chan<- SearchResult, opts SearchOptions) *resultProcessor {
	return &resultProcessor{
		results: results,
		seen:    make(map[uint64]bool),
		dedupe:  opts.DeduplicateFiles,
	}
}

func (rp *resultProcessor) add(result SearchResult) {
	if rp.dedupe {
		rp.mu.Lock()
		if rp.seen[result.Hash] {
			rp.mu.Unlock()
			return
		}
		rp.seen[result.Hash] = true
		rp.mu.Unlock()
	}
	rp.results <- result
}

func (rp *resultProcessor) close() {
	close(rp.results)
}

// calculateQuickHash generates a quick hash of file metadata and first few bytes
func calculateQuickHash(path string, info os.FileInfo, buf []byte) uint64 {
	h := xxhash.New()
	
	// Hash metadata
	h.Write([]byte(path))
	binary.Write(h, binary.LittleEndian, info.Size())
	binary.Write(h, binary.LittleEndian, info.ModTime().UnixNano())
	
	// Hash first few bytes of the file
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		if n, err := f.Read(buf[:1024]); err == nil {
			h.Write(buf[:n])
		}
	}
	
	return h.Sum64()
}

// newBatchProcessor creates a new batch processor
func newBatchProcessor(size int, callback func([]string)) *BatchProcessor {
	return &BatchProcessor{
		batch:    make([]string, 0, size),
		size:     size,
		callback: callback,
	}
}

func (bp *BatchProcessor) add(path string) {
	bp.batch = append(bp.batch, path)
	if len(bp.batch) >= bp.size {
		bp.flush()
	}
}

func (bp *BatchProcessor) flush() {
	if len(bp.batch) > 0 {
		bp.callback(bp.batch)
		bp.batch = bp.batch[:0]
	}
} 