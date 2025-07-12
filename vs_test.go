package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestFiles creates a temporary directory with test files
func setupTestFiles(t *testing.T) string {
	tmpDir := t.TempDir()
	
	// Create test video files
	testFiles := map[string][]byte{
		"test.mkv":     []byte("fake mkv content for testing range requests"),
		"movie.mp4":    []byte("fake mp4 content"),
		"video.avi":    []byte("fake avi content"),
		"sample.mov":   []byte("fake mov content"),
		"clip.wmv":     []byte("fake wmv content"),
		"stream.flv":   []byte("fake flv content"),
		"web.webm":     []byte("fake webm content"),
		"document.txt": []byte("not a video file"),
	}
	
	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	// Create subdirectory with video
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	subFile := filepath.Join(subDir, "sub.mkv")
	if err := os.WriteFile(subFile, []byte("subdirectory video"), 0644); err != nil {
		t.Fatalf("Failed to create subdirectory file: %v", err)
	}
	
	return tmpDir
}

func TestNewVideoServer(t *testing.T) {
	port := "8080"
	videoDir := "/test/dir"
	
	server := NewVideoServer(port, videoDir)
	
	if server.port != port {
		t.Errorf("Expected port %s, got %s", port, server.port)
	}
	
	if server.videoDir != videoDir {
		t.Errorf("Expected videoDir %s, got %s", videoDir, server.videoDir)
	}
}

func TestHandleRequest_FileNotFound(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	req := httptest.NewRequest("GET", "/nonexistent.mkv", nil)
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleRequest_DirectoryTraversalPrevention(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	// Test various directory traversal attempts
	traversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2fetc%2fpasswd",
	}
	
	for _, attempt := range traversalAttempts {
		req := httptest.NewRequest("GET", "/"+attempt, nil)
		rec := httptest.NewRecorder()
		
		server.handleRequest(rec, req)
		
		// Should return either Forbidden (403) or Not Found (404)
		// Both are acceptable as they prevent access
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
			t.Errorf("Directory traversal attempt '%s' should return %d or %d, got %d", 
				attempt, http.StatusForbidden, http.StatusNotFound, rec.Code)
		}
		
		// For forbidden, check the message
		if rec.Code == http.StatusForbidden && !strings.Contains(rec.Body.String(), "Access denied") {
			t.Errorf("Expected 'Access denied' message for traversal attempt '%s'", attempt)
		}
	}
}

func TestHandleRequest_FullFileServing(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	req := httptest.NewRequest("GET", "/test.mkv", nil)
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	
	expectedContent := "fake mkv content for testing range requests"
	if rec.Body.String() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, rec.Body.String())
	}
	
	// Check headers
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("Expected Accept-Ranges header to be 'bytes'")
	}
	
	contentType := rec.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header to be set")
	}
}

func TestHandleRangeRequest_ValidRange(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	// Test partial content request
	req := httptest.NewRequest("GET", "/test.mkv", nil)
	req.Header.Set("Range", "bytes=5-14")
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusPartialContent {
		t.Errorf("Expected status %d, got %d", http.StatusPartialContent, rec.Code)
	}
	
	expectedContent := "mkv conten"  // bytes 5-14 from "fake mkv content for testing range requests"
	if rec.Body.String() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, rec.Body.String())
	}
	
	// Check Content-Range header
	contentRange := rec.Header().Get("Content-Range")
	expectedRange := fmt.Sprintf("bytes 5-14/%d", len("fake mkv content for testing range requests"))
	if contentRange != expectedRange {
		t.Errorf("Expected Content-Range '%s', got '%s'", expectedRange, contentRange)
	}
}

func TestHandleRangeRequest_OpenEndedRange(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	// Test open-ended range (from byte 5 to end)
	req := httptest.NewRequest("GET", "/test.mkv", nil)
	req.Header.Set("Range", "bytes=5-")
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusPartialContent {
		t.Errorf("Expected status %d, got %d", http.StatusPartialContent, rec.Code)
	}
	
	expectedContent := "mkv content for testing range requests"  // from byte 5 to end
	if rec.Body.String() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, rec.Body.String())
	}
}

func TestHandleRangeRequest_InvalidRange(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	testCases := []struct {
		name       string
		rangeHeader string
		expectedStatus int
	}{
		{"Invalid format", "bytes=invalid", http.StatusBadRequest},
		{"Range beyond file size", "bytes=1000-2000", http.StatusRequestedRangeNotSatisfiable},
		{"Start greater than end", "bytes=10-5", http.StatusRequestedRangeNotSatisfiable},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test.mkv", nil)
			req.Header.Set("Range", tc.rangeHeader)
			rec := httptest.NewRecorder()
			
			server.handleRequest(rec, req)
			
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestListFiles_RootPath(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	
	body := rec.Body.String()
	
	// Check that HTML content is returned
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("Expected HTML response")
	}
	
	if !strings.Contains(body, "Available Videos") {
		t.Error("Expected 'Available Videos' title")
	}
	
	// Check that video files are listed
	videoFiles := []string{"test.mkv", "movie.mp4", "video.avi", "sample.mov", "clip.wmv", "stream.flv", "web.webm"}
	for _, file := range videoFiles {
		if !strings.Contains(body, file) {
			t.Errorf("Expected video file '%s' to be listed", file)
		}
	}
	
	// Check that non-video files are NOT listed
	if strings.Contains(body, "document.txt") {
		t.Error("Non-video file should not be listed")
	}
	
	// Check that subdirectory video is listed
	if !strings.Contains(body, "subdir/sub.mkv") || !strings.Contains(body, "subdir\\sub.mkv") {
		// Account for different path separators
		if !strings.Contains(body, "sub.mkv") {
			t.Error("Expected subdirectory video file to be listed")
		}
	}
}

func TestListFiles_ContentType(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type 'text/html', got '%s'", contentType)
	}
}

func TestMimeTypeDetection(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	testCases := []struct {
		filename      string
		acceptedMimes []string // Multiple acceptable MIME types for cross-platform compatibility
	}{
		{"test.mkv", []string{"video/x-matroska", "application/octet-stream"}}, // mkv varies by system
		{"movie.mp4", []string{"video/mp4"}},
		{"video.avi", []string{"video/x-msvideo", "video/avi", "video/vnd.avi", "application/octet-stream"}}, // AVI varies by system
		{"sample.mov", []string{"video/quicktime"}},
	}
	
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tc.filename, nil)
			rec := httptest.NewRecorder()
			
			server.handleRequest(rec, req)
			
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
			}
			
			contentType := rec.Header().Get("Content-Type")
			
			// Ensure some content type is set
			if contentType == "" {
				t.Error("Expected Content-Type header to be set")
				return
			}
			
			// Check if the detected MIME type is one of the accepted types
			mimeAccepted := false
			for _, acceptedMime := range tc.acceptedMimes {
				if contentType == acceptedMime {
					mimeAccepted = true
					break
				}
			}
			
			if !mimeAccepted {
				t.Errorf("Content-Type '%s' not in accepted types %v for file %s", 
					contentType, tc.acceptedMimes, tc.filename)
			}
		})
	}
}

func TestSubdirectoryAccess(t *testing.T) {
	tmpDir := setupTestFiles(t)
	server := NewVideoServer("8080", tmpDir)
	
	req := httptest.NewRequest("GET", "/subdir/sub.mkv", nil)
	rec := httptest.NewRecorder()
	
	server.handleRequest(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	
	expectedContent := "subdirectory video"
	if rec.Body.String() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, rec.Body.String())
	}
}

// Benchmark tests for performance
func BenchmarkHandleRequest_FullFile(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Create a larger test file
	testFile := filepath.Join(tmpDir, "large.mkv")
	content := strings.Repeat("test data ", 10000) // ~90KB
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	server := NewVideoServer("8080", tmpDir)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/large.mkv", nil)
		rec := httptest.NewRecorder()
		server.handleRequest(rec, req)
	}
}

func BenchmarkHandleRangeRequest(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Create a larger test file
	testFile := filepath.Join(tmpDir, "large.mkv")
	content := strings.Repeat("test data ", 10000) // ~90KB
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	server := NewVideoServer("8080", tmpDir)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/large.mkv", nil)
		req.Header.Set("Range", "bytes=1000-2000")
		rec := httptest.NewRecorder()
		server.handleRequest(rec, req)
	}
}

func BenchmarkListFiles(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Create many test files
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("video%d.mkv", i)
		testFile := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	server := NewVideoServer("8080", tmpDir)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.handleRequest(rec, req)
	}
}