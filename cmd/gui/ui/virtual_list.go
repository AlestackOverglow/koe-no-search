package ui

import (
	"fmt"
	"sync"
	"time"
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/container"
)

// CachedFileItem represents a file item with cached display string
type CachedFileItem struct {
	Path        string
	Size        int64
	DisplayText string
}

// SimpleCache provides a basic thread-safe cache
type SimpleCache struct {
	items map[int]string
	mutex sync.RWMutex
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		items: make(map[int]string, 1000),
	}
}

func (c *SimpleCache) Get(key int) (string, bool) {
	c.mutex.RLock()
	val, exists := c.items[key]
	c.mutex.RUnlock()
	return val, exists
}

func (c *SimpleCache) Put(key int, value string) {
	c.mutex.Lock()
	if value == "" {
		delete(c.items, key)
	} else {
		c.items[key] = value
	}
	// Clear cache if it gets too large
	if len(c.items) > 2000 {
		c.items = make(map[int]string, 1000)
	}
	c.mutex.Unlock()
}

func (c *SimpleCache) Remove(key int) {
	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()
}

func (c *SimpleCache) Clear() {
	c.mutex.Lock()
	c.items = make(map[int]string, 1000)
	c.mutex.Unlock()
}

// VirtualFileList is an optimized list widget for displaying large numbers of files
type VirtualFileList struct {
	widget.BaseWidget
	list        *widget.List
	items       *[]FileListItem
	cache       *SimpleCache
	scroller    *container.Scroll
	resultBuf   *ResultBuffer
	
	// Context for goroutine management
	ctx         context.Context
	cancelFunc  context.CancelFunc
	
	// Update batching
	updateQueue   chan updateRequest
	queueMutex    sync.Mutex
	isUpdating    bool
	
	// Scroll debouncing
	scrollTimer   *time.Timer
	lastScrollPos float32
	
	// Visible range tracking
	visibleStart int
	visibleEnd   int
	
	// Callbacks
	onSelected    func(id int)

	// Performance settings
	maxCacheSize     int
	visibleBuffer    int
	updateBatchSize  int
	updateInterval   time.Duration
}

type updateRequest struct {
	start, end int
	force      bool
}

// NewVirtualFileList creates a new virtual file list
func NewVirtualFileList(items *[]FileListItem) *VirtualFileList {
	ctx, cancel := context.WithCancel(context.Background())
	
	vlist := &VirtualFileList{
		items:        items,
		cache:        NewSimpleCache(),
		updateQueue:  make(chan updateRequest, 2000),
		ctx:          ctx,
		cancelFunc:   cancel,
		
		// Performance settings
		maxCacheSize:    10000,   // Increased for better performance
		visibleBuffer:   100,     // Increased buffer zone
		updateBatchSize: 100,     // Number of items to update at once
		updateInterval:  50 * time.Millisecond,
	}
	
	// Initialize base list with optimized update function
	list := widget.NewList(
		func() int {
			if vlist.resultBuf != nil {
				return vlist.resultBuf.GetTotalItems()
			}
			return len(*items)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextTruncate
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Alignment = fyne.TextAlignLeading
			label.Resize(fyne.NewSize(0, 40))
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			
			// Try to get from cache first
			if text, exists := vlist.cache.Get(int(i)); exists && text != "" {
				label.SetText(text)
				return
			}
			
			// Get item efficiently
			var item FileListItem
			var ok bool
			
			if vlist.resultBuf != nil {
				item, ok = vlist.resultBuf.GetItem(int(i))
			} else if i < len(*vlist.items) {
				item = (*vlist.items)[i]
				ok = true
			}
			
			if !ok {
				label.SetText("")
				return
			}
			
			// Format and cache the item
			displayText := vlist.formatFileItem(item)
			vlist.cache.Put(int(i), displayText)
			label.SetText(displayText)
		},
	)
	
	// Optimized scroll handler
	vlist.list = list
	vlist.scroller = container.NewScroll(list)
	vlist.scroller.OnScrolled = func(p fyne.Position) {
		viewportHeight := vlist.scroller.Size().Height
		itemHeight := float32(40)
		scrollPos := p.Y
		
		// Calculate visible range
		start := int(scrollPos / itemHeight)
		end := int((scrollPos + viewportHeight) / itemHeight) + 1
		
		// Update visible range with buffer
		newStart := max(0, start-vlist.visibleBuffer)
		var newEnd int
		if vlist.resultBuf != nil {
			newEnd = min(vlist.resultBuf.GetTotalItems(), end+vlist.visibleBuffer)
		} else {
			newEnd = min(len(*vlist.items), end+vlist.visibleBuffer)
		}
		
		// Update visible range and trigger refresh
		vlist.visibleStart = newStart
		vlist.visibleEnd = newEnd
		
		// Pre-cache items in the new range
		go vlist.preCacheRange(newStart, newEnd)
		
		// Refresh visible items
		if vlist.scrollTimer != nil {
			vlist.scrollTimer.Stop()
		}
		vlist.scrollTimer = time.AfterFunc(50*time.Millisecond, func() {
			vlist.refreshVisible()
		})
	}
	
	// Start update processor
	go vlist.processUpdates()
	
	vlist.ExtendBaseWidget(vlist)
	return vlist
}

// preCacheRange pre-caches items in the specified range
func (v *VirtualFileList) preCacheRange(start, end int) {
	for i := start; i < end; i++ {
		// Skip if already cached
		if _, exists := v.cache.Get(i); exists {
			continue
		}
		
		// Get and cache item
		var item FileListItem
		var ok bool
		
		if v.resultBuf != nil {
			item, ok = v.resultBuf.GetItem(i)
		} else if i < len(*v.items) {
			item = (*v.items)[i]
			ok = true
		}
		
		if ok {
			displayText := v.formatFileItem(item)
			v.cache.Put(i, displayText)
		}
	}
}

// formatFileItem formats a file item for display
func (v *VirtualFileList) formatFileItem(item FileListItem) string {
	var sizeStr string
	switch {
	case item.Size >= 1024*1024*1024:
		sizeStr = fmt.Sprintf("%.1f GB", float64(item.Size)/(1024*1024*1024))
	case item.Size >= 1024*1024:
		sizeStr = fmt.Sprintf("%.1f MB", float64(item.Size)/(1024*1024))
	case item.Size >= 1024:
		sizeStr = fmt.Sprintf("%.1f KB", float64(item.Size)/1024)
	default:
		sizeStr = fmt.Sprintf("%d B", item.Size)
	}
	
	return fmt.Sprintf("%s (%s)", item.Path, sizeStr)
}

// refreshVisible refreshes only the visible portion efficiently
func (v *VirtualFileList) refreshVisible() {
	if v.list == nil {
		return
	}
	v.list.Refresh()
}

// processUpdates handles batched updates efficiently
func (v *VirtualFileList) processUpdates() {
	batchTimer := time.NewTicker(v.updateInterval)
	defer batchTimer.Stop()
	
	var pendingUpdates []updateRequest
	
	for {
		select {
		case <-v.ctx.Done():
			return
			
		case req := <-v.updateQueue:
			pendingUpdates = append(pendingUpdates, req)
			
			// Process immediately if we have enough updates
			if len(pendingUpdates) >= v.updateBatchSize {
				v.processBatch(pendingUpdates)
				pendingUpdates = pendingUpdates[:0]
			}
			
		case <-batchTimer.C:
			if len(pendingUpdates) > 0 {
				v.processBatch(pendingUpdates)
				pendingUpdates = pendingUpdates[:0]
			}
		}
	}
}

// processBatch processes updates efficiently
func (v *VirtualFileList) processBatch(updates []updateRequest) {
	v.queueMutex.Lock()
	defer v.queueMutex.Unlock()
	
	if v.isUpdating {
		return
	}
	v.isUpdating = true
	
	// Process all updates
	for _, update := range updates {
		// Pre-cache items in the updated range
		go v.preCacheRange(update.start, update.end)
	}
	
	// Clear cache for items far outside visible range
	v.cache.mutex.Lock()
	for k := range v.cache.items {
		if k < v.visibleStart-v.visibleBuffer*3 || k > v.visibleEnd+v.visibleBuffer*3 {
			delete(v.cache.items, k)
		}
	}
	v.cache.mutex.Unlock()
	
	// Refresh visible items
	v.refreshVisible()
	
	v.isUpdating = false
}

// SetResultBuffer sets the result buffer for this list
func (v *VirtualFileList) SetResultBuffer(rb *ResultBuffer) {
	v.resultBuf = rb
	v.Refresh()
}

// Cleanup releases resources
func (v *VirtualFileList) Cleanup() {
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	if v.scrollTimer != nil {
		v.scrollTimer.Stop()
	}
}

// CreateRenderer implements the fyne.Widget interface
func (v *VirtualFileList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.scroller)
}

// MinSize implements the fyne.Widget interface
func (v *VirtualFileList) MinSize() fyne.Size {
	return v.scroller.MinSize()
}

// Resize implements the fyne.Widget interface
func (v *VirtualFileList) Resize(size fyne.Size) {
	v.BaseWidget.Resize(size)
	v.scroller.Resize(size)
}

// Move implements the fyne.Widget interface
func (v *VirtualFileList) Move(pos fyne.Position) {
	v.BaseWidget.Move(pos)
	v.scroller.Move(pos)
}

// SetOnSelected sets the callback for when an item is selected
func (v *VirtualFileList) SetOnSelected(callback func(id int)) {
	v.onSelected = callback
	v.list.OnSelected = func(id widget.ListItemID) {
		if v.onSelected != nil {
			v.onSelected(int(id))
		}
		v.list.UnselectAll()
	}
}

// Refresh refreshes the list and clears the cache
func (v *VirtualFileList) Refresh() {
	v.cache.Clear()
	v.list.Refresh()
}

// UnselectAll removes any selection in the list
func (v *VirtualFileList) UnselectAll() {
	v.list.UnselectAll()
}

// RefreshWithRange queues a range update
func (v *VirtualFileList) RefreshWithRange(start, end int) {
	select {
	case v.updateQueue <- updateRequest{start: start, end: end, force: false}:
		// Update queued successfully
	default:
		// Queue is full, force update
		select {
		case v.updateQueue <- updateRequest{start: start, end: end, force: true}:
			// Forced update queued
		default:
			// Even force queue failed, do immediate refresh
			if end >= v.visibleStart && start <= v.visibleEnd {
				v.cache.mutex.Lock()
				for i := max(start, v.visibleStart); i < min(end, v.visibleEnd); i++ {
					delete(v.cache.items, i)
				}
				v.cache.mutex.Unlock()
				v.refreshVisible()
			}
		}
	}
	
	// Also refresh the list to update the total count
	v.list.Refresh()
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// AsListWidget returns the list as a CanvasObject for use in the UI
func (v *VirtualFileList) AsListWidget() fyne.CanvasObject {
	return v.scroller
} 