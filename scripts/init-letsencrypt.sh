#!/bin/bash
# ===========================================
# Let's Encrypt SSL Certificate Initialization
# ===========================================
# Run this script once on initial deployment to obtain SSL certificates

set -euo pipefail

# Configuration
DOMAIN="${DOMAIN:-gisty.co}"
EMAIL="${EMAIL:-admin@gisty.co}"
STAGING="${STAGING:-0}"  # Set to 1 for testing

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
CERTBOT_PATH="${PROJECT_DIR}/certbot"
NGINX_PATH="${PROJECT_DIR}/nginx"

echo "=== Let's Encrypt SSL Certificate Setup ==="
echo "Domain: ${DOMAIN}"
echo "Email: ${EMAIL}"
echo "Staging: ${STAGING}"
echo ""

# Create certbot directories
mkdir -p "${CERTBOT_PATH}/conf"
mkdir -p "${CERTBOT_PATH}/www"

# Download recommended TLS parameters
if [ ! -e "${CERTBOT_PATH}/conf/options-ssl-nginx.conf" ]; then
    echo "Downloading recommended TLS parameters..."
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "${CERTBOT_PATH}/conf/options-ssl-nginx.conf"
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "${CERTBOT_PATH}/conf/ssl-dhparams.pem"
fi

cd "${PROJECT_DIR}"

# Step 1: Use init config (HTTP only) to start nginx and get certificate
echo "Step 1: Starting nginx with HTTP-only config..."
cp "${NGINX_PATH}/nginx.init.conf" "${NGINX_PATH}/nginx.conf.bak"
cp "${NGINX_PATH}/nginx.init.conf" "${NGINX_PATH}/nginx.conf"

# Start all services
docker compose -f docker-compose.prod.yml --env-file deploy/.env up -d

echo "Waiting for services to start..."
sleep 10

# Step 2: Request certificate
echo "Step 2: Requesting Let's Encrypt certificate..."

STAGING_ARG=""
if [ "$STAGING" = "1" ]; then
    STAGING_ARG="--staging"
fi

docker compose -f docker-compose.prod.yml run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    ${STAGING_ARG} \
    --email ${EMAIL} \
    --agree-tos \
    --no-eff-email \
    -d ${DOMAIN} \
    -d www.${DOMAIN}" certbot

# Step 3: Switch to full SSL config
echo "Step 3: Switching to SSL config..."
# Restore the original nginx.conf (with SSL)
git checkout "${NGINX_PATH}/nginx.conf" 2>/dev/null || cp "${NGINX_PATH}/nginx.conf.bak" "${NGINX_PATH}/nginx.conf"

# Reload nginx with SSL config
docker compose -f docker-compose.prod.yml exec nginx nginx -s reload

# Cleanup
rm -f "${NGINX_PATH}/nginx.conf.bak"

echo ""
echo "=== SSL Certificate Setup Complete ==="
echo "Certificate obtained for: ${DOMAIN}"
echo ""
echo "To renew certificates, run:"
echo "  docker compose -f docker-compose.prod.yml run --rm certbot renew"
