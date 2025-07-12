# Video Server (vs) Deployment

Secure deployment configuration for the vs video streaming server with Nginx reverse proxy.

## Quick Start

1. **Prepare video directory:**
   ```bash
   mkdir -p videos
   # Copy your video files to videos/
   ```

2. **Deploy with Docker:**
   ```bash
   cd deployment/docker
   docker-compose up -d
   ```

3. **Access:**
   - Web interface: http://localhost
   - Health check: http://localhost/health

## HTTPS Deployment

1. **Generate SSL certificates:**
   ```bash
   ./deployment/scripts/generate-ssl.sh
   ```

2. **Update docker-compose.yml to use HTTPS config:**
   ```yaml
   nginx:
     volumes:
       - ../nginx/nginx-ssl.conf:/etc/nginx/nginx.conf:ro
   ```

3. **Deploy:**
   ```bash
   cd deployment/docker
   docker-compose up -d
   ```

## Configuration

### Rate Limiting
- **API requests**: 10/second (burst: 20)
- **Video streaming**: 5/second (burst: 10)

Adjust in nginx config files as needed for your environment.

### Security
- Backend isolated in Docker network (port 32767 not exposed)
- Only Nginx proxy exposed (ports 80/443)
- Security headers included for video streaming

### Logs
- Access logs: `/var/log/nginx/vs_access.log`
- Error logs: `/var/log/nginx/vs_error.log`

## File Structure

```
deployment/
├── README.md                 # This file
├── docker/
│   ├── Dockerfile           # vs container build
│   └── docker-compose.yml   # Complete stack
├── nginx/
│   ├── nginx.conf          # HTTP proxy config
│   └── nginx-ssl.conf      # HTTPS proxy config
└── scripts/
    └── generate-ssl.sh     # SSL certificate utility
```