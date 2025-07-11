package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type VideoServer struct {
	port     string
	videoDir string
}

func NewVideoServer(port, videoDir string) *VideoServer {
	return &VideoServer{
		port:     port,
		videoDir: videoDir,
	}
}

func (vs *VideoServer) Start() {
	http.HandleFunc("/", vs.handleRequest)
	
	fmt.Printf("Starting video streaming server on port %s\n", vs.port)
	fmt.Printf("Serving files from: %s\n", vs.videoDir)
	fmt.Printf("Access videos at: http://0.0.0.0:%s/filename.mkv\n", vs.port)
	
	log.Fatal(http.ListenAndServe(":"+vs.port, nil))
}

func (vs *VideoServer) handleRequest(w http.ResponseWriter, r *http.Request) {
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
	videoDir := "."
	port := "32767"
	
	// Parse command line arguments
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
