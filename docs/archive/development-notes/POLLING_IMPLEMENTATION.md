# Efficient Paginated Polling Implementation

## Overview

This document describes the implementation of an efficient paginated polling system for the Jobs page, replacing the SSE (Server-Sent Events) approach with a more scalable solution.

## Key Features

### 1. **Paginated Data Fetching**
- Only fetches data for the current page (e.g., 25 jobs per page)
- Reduces network traffic and server load
- Faster response times for large job lists

### 2. **Advanced Filtering**
- **Status Filter**: Filter by pending, running, completed, failed, or interrupted
- **Priority Filter**: Filter by priority levels (1-5)
- **Search**: Search in job names and hashlist names
- All filters work with pagination

### 3. **Smart Polling**
- Polls every 5 seconds by default
- Only fetches current page with active filters
- Can be toggled on/off by users
- Cancels in-flight requests when parameters change

### 4. **Status Counts**
- Shows badge counts for each status
- Updates with each poll
- Helps users understand job distribution

## Backend Implementation

### Repository Layer (`job_execution_repository_extension.go`)

```go
// New filter structure
type JobFilter struct {
    Status   *string
    Priority *int
    Search   *string
}

// Key methods added:
- ListWithFilters()      // Paginated list with filters
- GetFilteredCount()     // Count matching filter criteria
- GetStatusCounts()      // Get counts by status
```

### Handler Layer (`user_jobs.go`)

The `/api/jobs` endpoint now supports:
- `page` - Page number (default: 1)
- `page_size` - Items per page (default: 25, max: 200)
- `status` - Filter by job status
- `priority` - Filter by priority (1-5)
- `search` - Search in job/hashlist names

Response includes:
```json
{
  "jobs": [...],
  "pagination": {
    "page": 1,
    "page_size": 25,
    "total": 150,
    "total_pages": 6
  },
  "status_counts": {
    "pending": 10,
    "running": 5,
    "completed": 120,
    "failed": 10,
    "interrupted": 5
  }
}
```

## Frontend Implementation

### Jobs Page (`/frontend/src/pages/Jobs/index.tsx`)

Key features:
1. **State Management**
   - Separate state for pagination, filters, and data
   - Maintains user's position during polls

2. **Efficient Polling**
   - Uses `setInterval` with 5-second intervals
   - Cancels previous requests using `AbortController`
   - Only shows loading on initial load or manual refresh

3. **User Controls**
   - Toggle auto-refresh on/off
   - Manual refresh button
   - Page size selector
   - Status filter buttons with counts
   - Priority dropdown filter
   - Search field

4. **Performance Optimizations**
   - Debounced search input
   - Memoized query building
   - Proper cleanup on unmount

## Benefits Over SSE

1. **Scalability**
   - No persistent connections
   - Reduced server memory usage
   - Works better with load balancers

2. **Efficiency**
   - Only fetches visible data
   - Reduces bandwidth usage
   - Faster initial page loads

3. **User Experience**
   - Maintains user's current view
   - No sudden jumps or resets
   - Clear feedback on data freshness

4. **Reliability**
   - No connection drops
   - Works with all proxies
   - Simpler error handling

## Usage Examples

### Basic Usage
1. Navigate to Jobs page
2. Jobs auto-refresh every 5 seconds
3. Use pagination to navigate large lists

### Filtering
1. Click status buttons to filter by status
2. Select priority from dropdown
3. Type in search box to search jobs

### Performance Tuning
1. Increase page size for fewer requests
2. Disable auto-refresh when not needed
3. Use filters to reduce data volume

## Migration Notes

To migrate from SSE to polling:
1. Update backend handlers to support filtering
2. Replace SSE hooks with polling logic
3. Add filter UI components
4. Test with large datasets

## Future Enhancements

1. **Configurable Poll Interval**: Allow users to set custom intervals
2. **Smart Polling**: Slow down polling when no changes detected
3. **Batch Operations**: Select multiple jobs for bulk actions
4. **Export Functionality**: Export filtered job lists
5. **Advanced Filters**: Date ranges, agent filters, etc.