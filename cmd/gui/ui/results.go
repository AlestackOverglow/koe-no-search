package ui

import (
	"filesearch/cmd/gui/explorer"
	"sync"
	"time"
)

// FileListItem represents an item in the file list
type FileListItem struct {
	Path string
	Size int64
}

// ResultBuffer handles buffered updates of search results
type ResultBuffer struct {
	items      []FileListItem
	mu         sync.Mutex
	foundFiles *[]FileListItem
	list       *VirtualFileList
	lastCount  int
	lastUpdate time.Time
	
	// Pre-allocated buffers
	batchBuffer []FileListItem
	
	// Chunked storage
	chunks     [][]FileListItem
	chunkSize  int
	totalItems int

	// Update thresholds
	minUpdateInterval time.Duration
	batchSize        int
}

// NewResultBuffer creates a new buffer with adaptive capacity
func NewResultBuffer(foundFiles *[]FileListItem, list *VirtualFileList) *ResultBuffer {
	initialCapacity := 10000
	chunkSize := 50000
	
	*foundFiles = make([]FileListItem, 0, initialCapacity)
	
	rb := &ResultBuffer{
		items:       make([]FileListItem, 0, 1000),
		foundFiles:  foundFiles,
		list:        list,
		lastCount:   0,
		lastUpdate:  time.Now(),
		batchBuffer: make([]FileListItem, 0, 2000),
		chunks:      make([][]FileListItem, 0, 100),
		chunkSize:   chunkSize,
		
		// Update thresholds
		minUpdateInterval: 100 * time.Millisecond,  // Decreased minimum time between updates
		batchSize:        1000,                     // Minimum items for update
	}
	
	list.SetResultBuffer(rb)
	return rb
}

// Add adds a new item to the buffer and flushes if needed
func (rb *ResultBuffer) Add(item FileListItem) {
	rb.mu.Lock()
	
	// Append to items slice
	rb.items = append(rb.items, item)
	
	// Adaptive capacity scaling based on total items
	shouldFlush := len(rb.items) >= 1000 // Fixed buffer size for better performance
	timeSinceLastUpdate := time.Since(rb.lastUpdate)
	
	rb.mu.Unlock()
	
	// Flush if buffer is full or enough time has passed
	if shouldFlush || timeSinceLastUpdate > time.Second {
		rb.Flush()
	}
}

// Flush updates the GUI with current buffer contents
func (rb *ResultBuffer) Flush() {
	rb.mu.Lock()
	if len(rb.items) == 0 {
		rb.mu.Unlock()
		return
	}
	
	// Reuse batch buffer
	if cap(rb.batchBuffer) < len(rb.items) {
		rb.batchBuffer = make([]FileListItem, 0, len(rb.items)*2)
	}
	rb.batchBuffer = rb.batchBuffer[:len(rb.items)]
	copy(rb.batchBuffer, rb.items)
	
	// Clear items slice without reallocating
	rb.items = rb.items[:0]
	currentCount := rb.totalItems
	
	// Add items to chunks
	batchItemCount := len(rb.batchBuffer)
	requiredChunks := (currentCount + batchItemCount + rb.chunkSize - 1) / rb.chunkSize
	
	// Pre-allocate chunks if needed
	if len(rb.chunks) < requiredChunks {
		newChunks := make([][]FileListItem, requiredChunks-len(rb.chunks))
		for i := range newChunks {
			newChunks[i] = make([]FileListItem, 0, rb.chunkSize)
		}
		rb.chunks = append(rb.chunks, newChunks...)
	}
	
	// Add items to chunks
	chunkIndex := currentCount / rb.chunkSize
	itemIndex := currentCount % rb.chunkSize
	
	for _, item := range rb.batchBuffer {
		if itemIndex == rb.chunkSize {
			chunkIndex++
			itemIndex = 0
		}
		rb.chunks[chunkIndex] = append(rb.chunks[chunkIndex], item)
		itemIndex++
		currentCount++
	}
	
	rb.totalItems = currentCount
	
	// Update foundFiles slice only if needed
	if len(*rb.foundFiles) < rb.totalItems {
		// Grow in larger chunks to reduce allocations
		newSize := ((rb.totalItems + rb.chunkSize - 1) / rb.chunkSize) * rb.chunkSize
		if cap(*rb.foundFiles) < newSize {
			newSlice := make([]FileListItem, rb.totalItems, newSize)
			copy(newSlice, *rb.foundFiles)
			*rb.foundFiles = newSlice
		} else {
			*rb.foundFiles = (*rb.foundFiles)[:rb.totalItems]
		}
		
		// Copy only new items
		destIndex := rb.lastCount
		for _, item := range rb.batchBuffer {
			(*rb.foundFiles)[destIndex] = item
			destIndex++
		}
	}
	
	rb.mu.Unlock()
	
	// Determine if UI update is needed
	timeSinceLastUpdate := time.Since(rb.lastUpdate)
	newItems := currentCount - rb.lastCount
	
	shouldUpdate := false
	switch {
	case currentCount < 100:
		// For very small sets, update immediately
		shouldUpdate = newItems > 0
	case currentCount < 1000:
		// For small sets, update frequently
		shouldUpdate = newItems > 0 && timeSinceLastUpdate >= 100*time.Millisecond
	case currentCount < 10000:
		// For medium sets, batch updates
		shouldUpdate = (newItems >= rb.batchSize && timeSinceLastUpdate >= rb.minUpdateInterval) ||
			timeSinceLastUpdate >= time.Second
	default:
		// For large sets, use larger batches and longer intervals
		shouldUpdate = (newItems >= rb.batchSize*2 && timeSinceLastUpdate >= rb.minUpdateInterval*2) ||
			timeSinceLastUpdate >= 2*time.Second
	}
	
	if shouldUpdate {
		rb.list.RefreshWithRange(rb.lastCount, currentCount)
		rb.lastCount = currentCount
		rb.lastUpdate = time.Now()
	}
}

// GetItem returns an item at the specified index
func (rb *ResultBuffer) GetItem(index int) (FileListItem, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if index < 0 || index >= rb.totalItems {
		return FileListItem{}, false
	}
	
	chunkIndex := index / rb.chunkSize
	itemIndex := index % rb.chunkSize
	
	if chunkIndex >= len(rb.chunks) || itemIndex >= len(rb.chunks[chunkIndex]) {
		return FileListItem{}, false
	}
	
	return rb.chunks[chunkIndex][itemIndex], true
}

// GetTotalItems returns the total number of items
func (rb *ResultBuffer) GetTotalItems() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.totalItems
}

// CreateResultsList creates and returns a list widget for search results
func CreateResultsList() (*VirtualFileList, *[]FileListItem) {
	var foundFiles []FileListItem
	vlist := NewVirtualFileList(&foundFiles)
	_ = NewResultBuffer(&foundFiles, vlist)
	
	vlist.SetOnSelected(func(id int) {
		if id < len(foundFiles) {
			go explorer.ShowInExplorer(foundFiles[id].Path)
		}
	})
	
	return vlist, &foundFiles
} 