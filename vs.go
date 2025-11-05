package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type VideoServer struct {
	port      string
	videoDir  string
	startTime time.Time
}

func NewVideoServer(port, videoDir string) *VideoServer {
	return &VideoServer{
		port:      port,
		videoDir:  videoDir,
		startTime: time.Now(),
	}
}

func (vs *VideoServer) Start() {
	// Set up HTTP handler with logging middleware
	http.HandleFunc("/", vs.loggingMiddleware(vs.handleRequest))

	// Create HTTP server
	server := &http.Server{
		Addr: ":" + vs.port,
	}

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting video streaming server on port %s\n", vs.port)
		fmt.Printf("Serving files from: %s\n", vs.videoDir)
		fmt.Printf("Access videos at: http://0.0.0.0:%s/filename.mkv\n", vs.port)
		fmt.Printf("Health check available at: http://0.0.0.0:%s/health\n", vs.port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("\nShutting down server gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server stopped")
	}
}

// loggingMiddleware wraps the handler with structured logging
func (vs *VideoServer) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code and bytes written
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next(lrw, r)

		// Log the request
		duration := time.Since(start)
		log.Printf("[%s] %s %s - %d - %d bytes - %v",
			start.Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			lrw.statusCode,
			lrw.bytesWritten,
			duration,
		)
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code and bytes written
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.bytesWritten += n
	return n, err
}

func (vs *VideoServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Handle health check endpoint
	if r.URL.Path == "/health" {
		uptime := time.Since(vs.startTime)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","uptime":"%s"}`, uptime.String())
		return
	}

	// Get file path from URL
	filePath := strings.TrimPrefix(r.URL.Path, "/")
	fullPath := filepath.Join(vs.videoDir, filePath)

	// Security check - prevent directory traversal
	absVideoDir, _ := filepath.Abs(vs.videoDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFullPath, absVideoDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if os.IsNotExist(err) || fileInfo.IsDir() {
		if r.URL.Path == "/" {
			vs.listFiles(w, r)
			return
		}
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get file info
	fileSize := fileInfo.Size()
	mimeType := mime.TypeByExtension(filepath.Ext(fullPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Set basic headers
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Accept-Ranges", "bytes")

	// Handle range requests for seeking support
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		vs.handleRangeRequest(w, r, fullPath, fileSize, rangeHeader)
	} else {
		// Serve entire file
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

		file, err := os.Open(fullPath)
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		io.Copy(w, file)
	}
}

func (vs *VideoServer) handleRangeRequest(w http.ResponseWriter, r *http.Request, filePath string, fileSize int64, rangeHeader string) {
	// Parse range header (e.g., "bytes=0-1023")
	re := regexp.MustCompile(`bytes=(\d+)-(\d*)`)
	matches := re.FindStringSubmatch(rangeHeader)

	if len(matches) < 3 {
		http.Error(w, "Invalid range header", http.StatusBadRequest)
		return
	}

	startByte, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid range start", http.StatusBadRequest)
		return
	}

	var endByte int64
	if matches[2] == "" {
		endByte = fileSize - 1
	} else {
		endByte, err = strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			http.Error(w, "Invalid range end", http.StatusBadRequest)
			return
		}
	}

	// Validate range
	if startByte >= fileSize || endByte >= fileSize || startByte > endByte {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		http.Error(w, "Range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := endByte - startByte + 1

	// Set partial content headers
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, fileSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	// Stream the requested range
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Seek to start position
	file.Seek(startByte, 0)

	// Stream in chunks to avoid loading large amounts into memory
	chunkSize := int64(1024 * 1024) // 1MB chunks
	remaining := contentLength

	for remaining > 0 {
		toRead := chunkSize
		if remaining < chunkSize {
			toRead = remaining
		}

		written, err := io.CopyN(w, file, toRead)
		if err != nil && err != io.EOF {
			log.Printf("Error streaming file: %v", err)
			return
		}

		remaining -= written
		if written == 0 {
			break
		}
	}
}

func (vs *VideoServer) listFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Get video files
	videoExtensions := map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".avi":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
	}

	var files []FileInfo
	filepath.Walk(vs.videoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			relPath, _ := filepath.Rel(vs.videoDir, path)
			files = append(files, FileInfo{
				Name:     relPath,
				SizeMB:   float64(info.Size()) / 1024.0 / 1024.0,
				FullPath: path,
			})
		}
		return nil
	})

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Video Streaming Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .file-list { list-style: none; padding: 0; }
        .file-list li { margin: 10px 0; }
        .file-list a { 
            text-decoration: none; 
            color: #007bff; 
            font-size: 16px;
        }
        .file-list a:hover { text-decoration: underline; }
        .video-file { font-weight: bold; }
        .file-size { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <h1>Available Videos</h1>
    <ul class="file-list">
        {{range .}}
        <li>
            <a href="/{{.Name}}" class="video-file">{{.Name}}</a>
            <span class="file-size"> ({{printf "%.2f" .SizeMB}} MB)</span>
        </li>
        {{end}}
    </ul>
    <p style="color: #666; font-size: 14px; margin-top: 30px;">
        Copy the video URL and paste it into VLC: Media > Open Network Stream
    </p>
</body>
</html>`

	t := template.Must(template.New("files").Parse(tmpl))
	t.Execute(w, files)
}

type FileInfo struct {
	Name     string
	SizeMB   float64
	FullPath string
}

func main() {
	// Default values
	videoDir := "."
	port := "32767"

	// Check environment variables first
	if envPort := os.Getenv("VS_PORT"); envPort != "" {
		port = envPort
	}
	if envDir := os.Getenv("VS_DIR"); envDir != "" {
		videoDir = envDir
	}

	// Command line arguments override environment variables
	if len(os.Args) > 1 {
		videoDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	// Expand relative paths
	absVideoDir, err := filepath.Abs(videoDir)
	if err != nil {
		log.Fatalf("Error resolving video directory: %v", err)
	}

	// Check if video directory exists
	if _, err := os.Stat(absVideoDir); os.IsNotExist(err) {
		log.Fatalf("Video directory does not exist: %s", absVideoDir)
	}

	server := NewVideoServer(port, absVideoDir)
	server.Start()
}
