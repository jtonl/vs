#!/bin/bash

# SSL Certificate Generation Script
# Creates self-signed certificates for development/testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSL_DIR="$SCRIPT_DIR/../nginx/ssl"
DOMAIN="${1:-localhost}"

echo "=== SSL Certificate Generator ==="
echo "Domain: $DOMAIN"
echo "SSL Directory: $SSL_DIR"

# Create SSL directory
mkdir -p "$SSL_DIR"

# Generate private key
echo "Generating private key..."
openssl genrsa -out "$SSL_DIR/server.key" 2048

# Generate certificate signing request
echo "Generating certificate signing request..."
openssl req -new -key "$SSL_DIR/server.key" -out "$SSL_DIR/server.csr" -subj "/C=US/ST=CA/L=San Francisco/O=Video Server/OU=IT Department/CN=$DOMAIN"

# Generate self-signed certificate
echo "Generating self-signed certificate..."
openssl x509 -req -days 365 -in "$SSL_DIR/server.csr" -signkey "$SSL_DIR/server.key" -out "$SSL_DIR/server.crt" -extensions v3_req -extfile <(
cat <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# Set proper permissions
chmod 600 "$SSL_DIR/server.key"
chmod 644 "$SSL_DIR/server.crt"

# Clean up CSR
rm "$SSL_DIR/server.csr"

echo "✅ SSL certificates generated successfully!"
echo "Certificate: $SSL_DIR/server.crt"
echo "Private Key: $SSL_DIR/server.key"
echo ""
echo "To use HTTPS, update docker-compose.yml to mount nginx-ssl.conf:"
echo "  nginx:"
echo "    volumes:"
echo "      - ./nginx/nginx-ssl.conf:/etc/nginx/nginx.conf:ro"
echo ""
echo "Certificate details:"
openssl x509 -in "$SSL_DIR/server.crt" -text -noout | grep -A 1 "Subject:"
openssl x509 -in "$SSL_DIR/server.crt" -text -noout | grep -A 5 "X509v3 Subject Alternative Name"