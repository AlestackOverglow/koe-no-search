# Koe no Search - Fast File Search Application

## Description
Koe no Search is a modern file search application for computers. It combines powerful search algorithms with a user-friendly graphical interface, making it suitable for both regular users and professionals.

## Key Features
- Fast file search by name and extension
- Filtering by file size and age
- Regular expression support
- Symbolic link handling
- Hidden file exclusion
- Duplicate removal
- File operations (copy, move, delete)

## How It Works

### 1. Search Algorithm
The program uses a multi-threaded file system traversal algorithm. Here's how it works:

1. **Thread Division**
   - Program determines available CPU cores
   - Creates corresponding number of worker threads
   - Each thread processes its part of the file system

2. **Smart Buffering**
   - Files are processed in batches of 1000
   - This reduces system load
   - Minimizes disk access operations

3. **Prioritization**
   - Some directories can have high priority
   - Others have low priority
   - This allows showing the most important results first

### 2. Memory Optimization
The program uses several techniques for efficient memory usage:

1. **Buffer Pool**
   - Limited number of buffers created for file reading
   - Buffers are reused
   - This prevents memory leaks

2. **Memory Mapping**
   - Used for large files (>1MB)
   - Allows working with files without loading them entirely into memory
   - Significantly speeds up processing of large files

### 3. Caching
The program implements several caching levels:

1. **Pattern Cache**
   - Regular expressions are compiled once
   - Stored for reuse
   - This speeds up filename checking

2. **Directory Cache**
   - Already checked directories are remembered
   - Information about whether to skip them is stored
   - Speeds up repeated searches

### 4. File Processing
The following optimizations are used when working with files:

1. **Smart Copying**
   - Buffer size adapts to file size
   - Direct I/O is used where possible
   - Atomic operations for reliability

2. **Conflict Handling**
   - Three strategies: skip, overwrite, rename
   - Timestamps are used for renaming
   - Random suffixes as a fallback

### 5. Logging
The program maintains detailed operation logs:

1. **Log Rotation**
   - Automatic creation of new files
   - Size limitation
   - Storage of several recent versions

2. **Record Buffering**
   - Logs are written in batches
   - This reduces disk load
   - Important messages (errors) are written immediately

## Technical Details

### Data Structures Used

1. **Hash Tables**
   - Used for fast search and comparison
   - Optimized for file path operations
   - Efficient in memory and access speed

2. **Priority Queues**
   - For processing files in the right order
   - Three priority levels
   - Adaptive size adjustment

## Detailed Algorithm Description

### 1. Parallel File System Traversal Algorithm

1. **Initial Phase**
   - Fixed batch size of 1000 files
   - Simple semaphore for concurrency control
   - Number of workers limited to CPU core count

2. **Directory Processing**
   - Each directory processed recursively
   - Files collected in batches
   - Immediate filtering of unwanted paths

3. **Batch Processing**
   - Batches sent through channels
   - Stop channel for graceful cancellation
   - Efficient memory reuse

4. **Skip Optimization**
   - Cache of skipped directories
   - Quick checks for hidden files
   - Efficient path prefix matching

### 2. File Filtering Algorithm

1. **Quick Preliminary Check**
   - Hidden file check based on dot prefix
   - Skip list for common system extensions
   - Simple string contains for pattern matching

2. **Pattern Optimization**
   - Separation of simple and regex patterns
   - Pattern compilation cache
   - Case-insensitive handling through lowercase cache

3. **Extension Processing**
   - Efficient extension extraction
   - Sorted list for binary search
   - Case normalization through cache

4. **Performance Optimizations**
   - Minimized string allocations
   - Reusable pattern cache
   - Early termination on first match

### 3. Duplicate Detection Algorithm

1. **Quick Hashing**
   - xxHash64 hash calculated for each file
   - Hash stored in result structure
   - Allows for quick file comparison

2. **Hash Comparison**
   - Files grouped by hash
   - Same hashes indicate potential duplicates
   - Efficient hash table used for grouping

3. **Memory Optimization**
   - Only hashes stored, not file contents
   - Hash takes only 8 bytes (uint64)
   - Allows efficient processing of large file sets

### 4. Result Prioritization Algorithm

1. **Channel-based Priority**
   - Three fixed-size channels (high, normal, low)
   - Each channel has 10,000 element capacity
   - Non-blocking send with overflow handling

2. **Directory-based Priority**
   - Priority determined by directory path
   - Simple prefix matching for priority dirs
   - Default to normal priority

3. **Processing Order**
   - High priority channel processed first
   - Normal priority as default flow
   - Low priority for specified paths

### 5. Smart Copy Algorithm

1. **Buffer Management**
   - Fixed buffer size (128KB default)
   - Buffer pool with maximum size limit
   - Automatic buffer reuse

2. **Copy Process**
   - Source file access check
   - Temporary file creation
   - Buffered copy with progress tracking

3. **Safety Measures**
   - Two-phase copy (temp file + rename)
   - Automatic cleanup on failure
   - Permission preservation

### 6. Log Rotation Algorithm

1. **Buffered Writing**
   - Buffered writer with 32KB size
   - Asynchronous channel with 1000 message capacity
   - Non-blocking message sending for regular logs

2. **Log Management**
   - Log directory in system temp folder
   - Maximum file size of 10MB
   - Initialization log entry with timestamp

3. **Error Handling**
   - Immediate flush for error messages
   - Graceful recovery from write failures
   - Fallback to disabled state on critical errors

### 7. Memory Management Algorithm

1. **Buffer Pool Implementation**
   - Atomic operations for thread safety
   - Maximum pool size of 32 buffers
   - Default buffer size of 128KB

2. **Buffer Optimization**
   - Minimum buffer size of 4KB
   - Maximum buffer size of 1MB
   - Size adaptation based on file size

3. **Resource Management**
   - Deferred cleanup using defer
   - Automatic buffer return to pool
   - Panic recovery in critical sections

### 8. Conflict Resolution Algorithm

1. **Conflict Detection**
   - Path existence check
   - Permission verification
   - Target directory space validation

2. **Resolution Strategies**
   - Skip: Returns empty path, no action taken
   - Overwrite: Returns original path for direct replacement
   - Rename: Implements progressive naming strategy

3. **Rename Implementation**
   - Up to 100 attempts with timestamp-based names
   - Random 8-byte suffix as fallback
   - Timestamp-only suffix as final fallback
   - Format: base_timestamp_counter.ext or base_random.ext

## Usage Recommendations

### Optimal Settings

1. **Thread Count**
   - For SSD: can use more threads (CPU * 2)
   - For HDD: better to limit to CPU core count
   - For network drives: CPU / 2

2. **Buffer Size**
   - For normal search: 1000 files is sufficient
   - For large directories: can increase to 5000
   - For limited memory: reduce to 500

3. **Memory Mapping**
   - Enable for large file searches
   - Disable when searching only by names
   - Set 1MB threshold for small files

### Search Tips

1. **Pattern Usage**
   - `*.txt` - all text files
   - `doc*.docx` - Word documents starting with "doc"
   - `*.{jpg,png,gif}` - any images of specified formats

2. **Filtering**
   - By size: "1MB", "2.5GB"
   - By age: "7d" (days), "2w" (weeks), "1m" (month)
   - By type: can exclude hidden files

## Conclusion
Koe no Search is the result of careful optimization and thoughtful architecture. The program efficiently uses system resources, providing fast search even in large file systems. At the same time, it remains easy to use thanks to its intuitive interface.

## Future Improvements

### 1. Memory Optimization

1. **Adaptive Buffer Management**
   - Dynamic buffer size based on available system memory
   - Real-time memory pressure monitoring
   - Predictive memory allocation based on usage patterns
   - Smart buffer pooling with scaling

2. **Enhanced Memory Pool**
   - Multi-level buffer pools for different sizes
   - Automatic pool size adjustment
   - Memory usage statistics and optimization
   - Thread-local buffer caching

3. **Smart Caching System**
   - Frequently accessed path caching
   - Pattern compilation result caching
   - Directory structure caching
   - Automatic cache invalidation

4. **Resource Control**
   - Memory limit enforcement
   - Graceful degradation under pressure
   - System-wide resource monitoring
   - Adaptive resource allocation

### 2. Search Performance Enhancement

1. **On-the-fly Indexing**
   - Temporary index creation for frequent directories
   - Background index updates
   - Memory-efficient index structure
   - Automatic index cleanup

2. **Recent Search Caching**
   - Cache for recent search results
   - Incremental result updates
   - Smart cache invalidation
   - Memory-bounded result storage

3. **Predictive Search**
   - Search pattern analysis
   - Usage pattern learning
   - Suggestion system
   - Background pre-fetching

4. **Search Optimization**
   - Pattern optimization engine
   - Search strategy selection
   - Priority path processing
   - Parallel pattern matching

### 3. File Processing Enhancement

1. **Archive Support**
   - ZIP, RAR, 7Z file processing
   - Streaming archive reading
   - Password-protected archive handling
   - Temporary extraction management

2. **Document Processing**
   - PDF content extraction
   - Office document parsing
   - Text encoding detection
   - Content preview generation

3. **Metadata Processing**
   - Extended attribute handling
   - EXIF data extraction
   - File signature analysis
   - Custom metadata support

4. **Large File Handling**
   - Partial file loading
   - Streaming processing
   - Progress tracking
   - Memory-efficient processing

### 4. Network Capabilities

1. **Network Drive Support**
   - Optimized network protocols
   - Connection pooling
   - Automatic reconnection
   - Bandwidth optimization

2. **Distributed Search**
   - Local network discovery
   - Work distribution
   - Result aggregation
   - Node health monitoring

3. **Cloud Integration**
   - Settings synchronization
   - Search history sync
   - Cross-device operation
   - Secure data transfer

### 5. CLI Enhancement

1. **Interactive TUI Mode**
   - Real-time result display
   - Interactive filtering
   - Progress visualization
   - Keyboard shortcuts

2. **Output Formatting**
   - JSON output support
   - CSV export capability
   - Custom format templates
   - Machine-readable output

3. **Batch Processing**
   - Script mode support
   - Batch operation queuing
   - Error handling and recovery
   - Operation logging

### 6. Integration System

1. **Plugin Architecture**
   - Dynamic plugin loading
   - Plugin lifecycle management
   - Resource isolation
   - Version compatibility

2. **External API**
   - REST API interface
   - WebSocket real-time updates
   - Authentication system
   - Rate limiting

3. **IDE Integration**
   - VSCode extension
   - JetBrains plugin
   - Sublime Text package
   - Custom editor support

### 7. Localization Framework

1. **Multi-language Support**
   - Dynamic language switching
   - Resource bundle management
   - Font support for all scripts
   - RTL layout support

2. **Auto-detection**
   - System language detection
   - User preference learning
   - Regional format handling
   - Fallback mechanism

3. **Translation Management**
   - Community translation platform
   - Translation verification
   - String externalization
   - Context documentation

These improvements will be implemented in future versions to enhance the functionality, performance, and user experience of the application.