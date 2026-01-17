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

# Create dummy certificate for nginx to start
CERT_PATH="${CERTBOT_PATH}/conf/live/${DOMAIN}"
if [ ! -d "${CERT_PATH}" ]; then
    echo "Creating dummy certificate for ${DOMAIN}..."
    mkdir -p "${CERT_PATH}"
    openssl req -x509 -nodes -newkey rsa:4096 -days 1 \
        -keyout "${CERT_PATH}/privkey.pem" \
        -out "${CERT_PATH}/fullchain.pem" \
        -subj "/CN=localhost"
fi

# Start nginx with dummy certificate
echo "Starting nginx..."
cd "${PROJECT_DIR}"
docker compose -f docker-compose.prod.yml up -d nginx

echo "Waiting for nginx to start..."
sleep 5

# Delete dummy certificate
echo "Deleting dummy certificate..."
docker compose -f docker-compose.prod.yml run --rm --entrypoint "\
  rm -Rf /etc/letsencrypt/live/${DOMAIN} && \
  rm -Rf /etc/letsencrypt/archive/${DOMAIN} && \
  rm -Rf /etc/letsencrypt/renewal/${DOMAIN}.conf" certbot

# Request real certificate
echo "Requesting Let's Encrypt certificate..."

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
    --force-renewal \
    -d ${DOMAIN} \
    -d www.${DOMAIN}" certbot

echo "Reloading nginx..."
docker compose -f docker-compose.prod.yml exec nginx nginx -s reload

echo ""
echo "=== SSL Certificate Setup Complete ==="
echo "Certificate obtained for: ${DOMAIN}"
echo ""
echo "To renew certificates, run:"
echo "  docker compose -f docker-compose.prod.yml run --rm certbot renew"
