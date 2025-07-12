# Secure Deployment Guide

This guide provides instructions for deploying the video streaming server with security best practices using Nginx as a reverse proxy.

## Architecture Overview

```
Internet → Nginx (Port 80/443) → Video Server (Port 32767)
```

The deployment uses:
- **Nginx**: Reverse proxy with security headers, rate limiting, and SSL termination
- **Docker Compose**: Container orchestration for easy deployment
- **Security Headers**: OWASP recommended headers for web security
- **Rate Limiting**: DDoS protection and bandwidth management

## Quick Start

1. **Prepare your video directory:**
   ```bash
   mkdir -p videos
   # Copy your video files to the videos/ directory
   ```

2. **Deploy the stack:**
   ```bash
   docker-compose up -d
   ```

3. **Access the service:**
   - HTTP: http://localhost
   - Health check: http://localhost/health

## Security Features

### Rate Limiting
- **API endpoints**: 10 requests/second per IP (burst: 20)
- **Video files**: 5 requests/second per IP (burst: 10)
- Prevents DDoS and reduces bandwidth abuse

### Security Headers
- `X-Frame-Options`: Prevents clickjacking attacks
- `X-Content-Type-Options`: Prevents MIME sniffing
- `X-XSS-Protection`: Basic XSS protection
- `Content-Security-Policy`: Restricts resource loading
- `Referrer-Policy`: Controls referrer information

### Network Security
- Backend server isolated in private Docker network
- Only Nginx exposed to external traffic
- Health checks for service monitoring

### Container Security
- Non-root user execution
- Read-only video volume mounts
- Minimal Alpine Linux base images
- Security updates via image rebuilds

## SSL/TLS Configuration

### 1. Generate SSL Certificates

**Option A: Self-signed (Development)**
```bash
mkdir -p nginx/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx/ssl/server.key \
  -out nginx/ssl/server.crt \
  -subj "/CN=localhost"
```

**Option B: Let's Encrypt (Production)**
```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate (replace your-domain.com)
sudo certbot certonly --standalone -d your-domain.com

# Copy certificates
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/server.crt
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/server.key
```

### 2. Enable HTTPS in Nginx

Add to `nginx/nginx.conf` inside the `http` block:

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /etc/nginx/ssl/server.crt;
    ssl_certificate_key /etc/nginx/ssl/server.key;
    
    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Same location blocks as HTTP server...
}

# HTTP to HTTPS redirect
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

## Production Deployment

### Environment Variables
```bash
# Create .env file for production settings
cat > .env << EOF
VS_PORT=32767
VS_DIR=/videos
NGINX_HOST=your-domain.com
EOF
```

### Monitoring and Logging
```bash
# View logs
docker-compose logs -f nginx
docker-compose logs -f video-server

# Monitor health
watch curl -s http://localhost/health
```

### Backup and Maintenance
```bash
# Backup video files
tar -czf videos-backup-$(date +%Y%m%d).tar.gz videos/

# Update containers
docker-compose pull
docker-compose up -d --force-recreate

# Security updates
docker system prune -f
```

## Firewall Configuration

### UFW (Ubuntu)
```bash
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable
```

### iptables
```bash
# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow HTTP/HTTPS
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Block direct access to backend
iptables -A INPUT -p tcp --dport 32767 -j DROP
```

## Performance Tuning

### Nginx Worker Configuration
Add to `nginx.conf`:
```nginx
worker_processes auto;
worker_rlimit_nofile 8192;

events {
    worker_connections 4096;
    use epoll;
    multi_accept on;
}
```

### File System Optimization
```bash
# For video storage filesystem
mount -o noatime,nodiratime /dev/sdb1 /videos
```

## Troubleshooting

### Common Issues

1. **502 Bad Gateway**
   ```bash
   docker-compose logs video-server
   # Check if backend is running
   ```

2. **Rate Limiting Errors**
   ```bash
   # Adjust rates in nginx.conf
   limit_req_zone $binary_remote_addr zone=api:10m rate=20r/s;
   ```

3. **SSL Certificate Errors**
   ```bash
   # Verify certificate
   openssl x509 -in nginx/ssl/server.crt -text -noout
   ```

### Health Checks
```bash
# Test backend directly
curl http://localhost:32767/

# Test through proxy
curl http://localhost/health

# Test SSL
curl -k https://localhost/health
```

## Security Considerations

1. **Network Isolation**: Backend never exposed directly
2. **Principle of Least Privilege**: Non-root container execution
3. **Defense in Depth**: Multiple security layers (headers, rate limiting, SSL)
4. **Monitoring**: Health checks and logging for security events
5. **Updates**: Regular security updates for base images

## Production Checklist

- [ ] SSL certificates configured and valid
- [ ] Firewall rules applied
- [ ] Rate limiting configured appropriately
- [ ] Security headers validated
- [ ] Health monitoring setup
- [ ] Log rotation configured
- [ ] Backup strategy implemented
- [ ] Network isolation verified
- [ ] Non-root execution confirmed