# Video "vs" Server

A lightweight, high-performance HTTP video streaming server written in Go. Stream your video files directly to any media player that supports HTTP range requests - no transcoding required!

## 🚀 Features

- **Zero Transcoding**: Serves original video files directly for maximum quality and minimal CPU usage
- **Full Seeking Support**: HTTP range requests enable instant seeking to any timestamp
- **Multi-Format Support**: Works with MKV, MP4, AVI, MOV, WMV, FLV, WebM
- **Web File Browser**: Built-in HTML interface to browse available videos
- **High Performance**: Concurrent streaming with efficient 1MB chunked delivery
- **Security**: Directory traversal protection
- **Zero Dependencies**: Single binary using only Go standard library
- **Cross-Platform**: Linux, Windows, macOS support
- **Production Ready**: Comprehensive test suite with CI/CD validation

## 📥 Installation

### Download Binary (Recommended)
```bash
# Download for your platform from releases
wget https://github.com/jtonl/vs/releases/latest/vs
chmod +x vs
```

### Build from Source
```bash
git clone https://github.com/jtonl/vs.git
cd vs
go build vs.go

# Run tests to verify build
go test -v
```

## 🎬 Quick Start

### Simple Deployment
```bash
# Start server with default settings (port 32767, current directory)
./vs

# Specify video directory
./vs /path/to/your/movies

# Custom directory and port
./vs /path/to/movies 8080
```

### Secure Production Deployment
```bash
# Clone repository
git clone https://github.com/jtonl/vs.git
cd vs

# Create video directory and add your files
mkdir videos
# Copy your video files to videos/

# Generate SSL certificates
./scripts/generate-ssl.sh

# Deploy with Docker Compose (includes Nginx reverse proxy)
docker-compose up -d

# Access securely
open https://localhost
```

**What you get:**
- HTTPS with SSL/TLS encryption
- Rate limiting and DDoS protection
- Security headers (XSS, CSRF protection)
- Health monitoring at `/health`
- Network-isolated backend

## 📺 Usage

### With VLC Media Player
1. Start the server: `./vs`
2. Open VLC → **Media** → **Open Network Stream**
3. Enter URL: `http://your-server-ip:32767/your-video.mkv`
4. Enjoy instant seeking and full playback control!

### Web Browser
Navigate to `http://your-server-ip:32767/` to browse available videos and get direct streaming links.

### Command Line Examples
```bash
# Stream a specific movie
curl -H "Range: bytes=0-1048576" http://localhost:32767/movie.mkv

# Get video info
ffprobe http://localhost:32767/movie.mkv

# Download specific portion
wget --header="Range: bytes=1000000-2000000" http://localhost:32767/movie.mkv
```

## 🏗️ Architecture

Video "vs" Server implements HTTP/1.1 range requests (RFC 7233) for efficient video streaming:

```
Client Request:  GET /video.mkv HTTP/1.1
                Range: bytes=1048576-2097151

Server Response: HTTP/1.1 206 Partial Content
                Content-Range: bytes 1048576-2097151/4294967296
                Content-Length: 1048576
                [1MB video chunk]
```

This enables:
- **Instant seeking**: Jump to any timestamp without buffering from start
- **Bandwidth efficiency**: Only download requested portions
- **Multiple connections**: Different clients can request different ranges simultaneously

## 🆚 vs FFmpeg/Transcoding

| Feature | Video "vs" Server | FFmpeg Streaming |
|---------|-------------------|------------------|
| CPU Usage | Minimal | High (encoding) |
| Seeking | Instant | Restart required |
| Quality | Original | Transcoded |
| Startup Time | Immediate | Encoding delay |
| Bandwidth | Efficient | Fixed bitrate |
| Multiple Clients | ✅ | Limited |

## 🔧 Configuration

### Environment Variables
```bash
export VS_PORT=32767
export VS_DIR=/path/to/videos
./vs
```

### Systemd Service
```ini
[Unit]
Description=vs
After=network.target

[Service]
Type=simple
User=videoserver
ExecStart=/usr/local/bin/vs /opt/videos 32767
Restart=always

[Install]
WantedBy=multi-user.target
```

## 🛡️ Security

- **Directory traversal protection**: Prevents access outside specified video directory
- **No file uploads**: Read-only server, no write operations
- **No authentication**: Designed for trusted networks (add reverse proxy for auth)

### Production Security with Nginx

For production deployments, use the included secure configuration with Nginx reverse proxy:

```bash
# Quick secure deployment with Docker
./scripts/generate-ssl.sh
docker-compose up -d
```

**Security Features:**
- **Rate limiting**: 10 req/s API, 5 req/s video files
- **Security headers**: XSS, CSRF, clickjacking protection
- **SSL/TLS encryption**: HTTPS with modern ciphers
- **Network isolation**: Backend not directly exposed
- **Health monitoring**: Built-in health checks

See [DEPLOYMENT.md](DEPLOYMENT.md) for complete security deployment guide.

## 🧪 Testing & Quality

Comprehensive test suite covering:
- **HTTP Server**: Full file serving, error handling, status codes
- **Range Requests**: Valid/invalid ranges, partial content (RFC 7233)
- **Security**: Directory traversal prevention, access control
- **File Browser**: HTML generation, video filtering, subdirectories
- **Performance**: Benchmarks for serving, range requests, file listing
- **Concurrency**: Race condition detection and concurrent access safety

```bash
# Run full test suite
go test -v

# Run with race detection
go test -race -v

# Run benchmarks
go test -bench=. -v

# Run with coverage report
go test -cover -v
```

## 🚀 Performance

Benchmarked on modest hardware:
- **Concurrent streams**: 50+ simultaneous clients
- **Memory usage**: ~10MB base + minimal per connection
- **Throughput**: Limited only by network bandwidth
- **Latency**: Sub-millisecond response times

### Optimization Tips
```bash
# For high-traffic scenarios
ulimit -n 65536  # Increase file descriptor limit
echo 'net.core.somaxconn = 65536' >> /etc/sysctl.conf
```

## 📱 Compatible Players

- **VLC Media Player** (recommended)
- **MPV Player**
- **Web Browsers** (Chrome, Firefox, Safari)
- **Mobile Apps** (VLC Mobile, MX Player)
- **Smart TVs** (models supporting HTTP streaming)
- **Kodi/Plex** (as HTTP source)

### Why vs Plex/Jellyfin?
- **Lighter resource usage**: No database, no metadata scanning
- **Faster startup**: Immediate availability
- **Direct file access**: No library management needed
- **Simple deployment**: Single binary

## 📄 License

MIT License - see [LICENSE](LICENSE.md) file for details.

---

**Video "vs" Server** - When you need video streaming that just works! 🎥✨