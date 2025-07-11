# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a lightweight HTTP video streaming server written in Go that serves video files with HTTP range request support for seeking. The server provides direct file streaming without transcoding, making it efficient for bandwidth and CPU usage.

## Development Commands

### Build and Run
```bash
# Build the application
go build vs.go

# Run with default settings (port 32767, current directory)
./vs

# Run with custom video directory
./vs /path/to/your/movies

# Run with custom directory and port
./vs /path/to/movies 8080
```

### Testing
```bash
# Run Go tests (if any are added)
go test ./...

# Test streaming functionality
curl -H "Range: bytes=0-1048576" http://localhost:32767/test-video.mkv
```

### Go Module Management
```bash
# Initialize/update go.mod
go mod init vs
go mod tidy
```

### CI/CD
```bash
# GitHub Actions workflow runs on push/PR to master
# Tests across Go versions 1.18-1.21
# Cross-platform builds available as artifacts
```

### Code Quality
```bash
# Run linting and static analysis
go vet ./...
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

## Architecture

### Core Components

**VideoServer struct (vs.go:17-20)**: Main server structure containing port and video directory configuration.

**HTTP Handler (vs.go:39-91)**: Single handler function that:
- Implements directory traversal protection
- Serves file listing at root path
- Handles both full file requests and HTTP range requests
- Supports seeking through range request parsing

**Range Request Handler (vs.go:93-166)**: Implements HTTP/1.1 Range Requests (RFC 7233) for video seeking:
- Parses `Range: bytes=start-end` headers
- Streams 1MB chunks to avoid memory issues
- Enables instant seeking in media players

**File Browser (vs.go:168-237)**: HTML template-based file listing with:
- Video file filtering (.mkv, .mp4, .avi, .mov, .wmv, .flv, .webm)
- File size display
- Direct streaming links

### Key Features
- Zero-dependency Go implementation using only standard library
- Concurrent streaming support
- Memory-efficient chunked streaming (1MB chunks)
- Security through directory traversal prevention
- MIME type detection for proper content headers

## Environment Variables

```bash
# Set custom port (alternative to command line)
export VS_PORT=32767

# Set custom video directory (alternative to command line)
export VS_DIR=/path/to/videos
```

## Development Notes

- The server uses `filepath.Join()` and `filepath.Abs()` for secure path handling
- Range request parsing uses regex: `bytes=(\d+)-(\d*)`
- File streaming uses `io.CopyN()` for memory-efficient chunked transfer
- The web interface template is embedded directly in the code (vs.go:200-233)
- Supports both full file serving and partial content (HTTP 206) responses

## Security Considerations

- Directory traversal protection implemented via `filepath.Abs()` prefix checking
- Read-only server - no file upload or modification capabilities
- No authentication - intended for trusted network environments