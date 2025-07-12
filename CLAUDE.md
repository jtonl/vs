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
# Run all tests
go test -v

# Run tests with race detection
go test -race -v

# Run benchmarks
go test -bench=. -v

# Run tests with coverage
go test -cover -v

# Test specific functionality manually
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

# Alternative linters for Go ≤1.21 (staticcheck requires >1.21)
# Option 1: revive (modern golint replacement)
go install github.com/mgechev/revive@latest
revive ./...

# Option 2: golint (deprecated but functional)
go install golang.org/x/lint/golint@latest
golint ./...

# Additional checks
gofmt -l .
go mod verify
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

## Code Quality Assessment

**Overall Rating**: A- (Excellent) - Well-engineered production-ready application

### Key Strengths
- RFC 7233 compliant HTTP Range Request implementation for video seeking
- Memory-efficient 1MB chunked streaming prevents memory exhaustion
- Robust directory traversal protection using `filepath.Abs()` prefix checking
- Zero external dependencies - uses only Go standard library

### Priority Improvements (from CODE_REVIEW.md)
1. **Add graceful shutdown** - Handle SIGINT/SIGTERM for clean shutdown
2. **Enhanced request logging** - Add middleware for operational insights
3. **Basic rate limiting** - Prevent DoS attacks in production environments
4. **Health check endpoint** - Add `/health` for monitoring systems

### Production Considerations
- Currently designed for trusted network environments (no authentication)
- Stateless design enables horizontal scaling and load balancing
- Single binary deployment with minimal configuration requirements

## Build and Deployment
```bash
# Build for current platform
go build vs.go

# Build for specific platforms (based on CI configuration)
GOOS=linux GOARCH=amd64 go build -o vs-linux-amd64 vs.go
GOOS=windows GOARCH=amd64 go build -o vs-windows-amd64.exe vs.go
GOOS=darwin GOARCH=amd64 go build -o vs-darwin-amd64 vs.go
```

## Single File Architecture
This application consists of a single Go file (`vs.go`) with no external dependencies. Key implementation details:
- Main HTTP handler at vs.go:39-91 processes all requests
- Range request logic at vs.go:93-166 implements RFC 7233 for video seeking
- File browser template embedded at vs.go:200-233 for web interface
- Security via filepath.Abs() prefix checking at vs.go:44-50

## Test Coverage
The test suite (`vs_test.go`) provides comprehensive coverage:
- **HTTP Server Tests**: Full file serving, error handling, status codes
- **Range Request Tests**: Valid ranges, invalid ranges, open-ended ranges, partial content
- **Security Tests**: Directory traversal prevention, access control
- **File Listing Tests**: HTML generation, video file filtering, subdirectory support
- **MIME Type Tests**: Content-Type detection for various video formats
- **Performance Tests**: Benchmarks for file serving, range requests, and file listing
- **Race Condition Tests**: Concurrent access safety verification
