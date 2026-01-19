#!/bin/bash
# ===========================================
# Let's Encrypt SSL Certificate Initialization
# Using DNS-01 challenge with GoDaddy
# ===========================================

set -euo pipefail

# Configuration
DOMAIN="${DOMAIN:-gisty.co}"
EMAIL="${EMAIL:-admin@gisty.co}"
STAGING="${STAGING:-0}"  # Set to 1 for testing

# GoDaddy API credentials (required)
GODADDY_API_KEY="${GODADDY_API_KEY:-}"
GODADDY_API_SECRET="${GODADDY_API_SECRET:-}"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
CERTBOT_PATH="${PROJECT_DIR}/certbot"

echo "=== Let's Encrypt SSL Certificate Setup (GoDaddy DNS) ==="
echo "Domain: ${DOMAIN}"
echo "Email: ${EMAIL}"
echo "Staging: ${STAGING}"
echo ""

# Validate GoDaddy credentials
if [ -z "$GODADDY_API_KEY" ] || [ -z "$GODADDY_API_SECRET" ]; then
    echo "ERROR: GoDaddy API credentials required!"
    echo ""
    echo "Usage:"
    echo "  GODADDY_API_KEY=xxx GODADDY_API_SECRET=xxx DOMAIN=gisty.co EMAIL=you@email.com ./init-letsencrypt.sh"
    echo ""
    echo "Get your API keys from: https://developer.godaddy.com/keys"
    exit 1
fi

# Create certbot directories
mkdir -p "${CERTBOT_PATH}/conf"
mkdir -p "${CERTBOT_PATH}/www"
mkdir -p "${CERTBOT_PATH}/logs"

# Create GoDaddy DNS authenticator script
cat > "${CERTBOT_PATH}/godaddy-auth.sh" << 'AUTHEOF'
#!/bin/bash
# GoDaddy DNS Authenticator for Certbot

GODADDY_API_KEY="$GODADDY_API_KEY"
GODADDY_API_SECRET="$GODADDY_API_SECRET"

# Extract domain parts
CERTBOT_DOMAIN="${CERTBOT_DOMAIN}"
CERTBOT_VALIDATION="${CERTBOT_VALIDATION}"

# For wildcard or subdomain, we need the root domain
ROOT_DOMAIN=$(echo "$CERTBOT_DOMAIN" | awk -F. '{print $(NF-1)"."$NF}')

# Determine the record name
if [ "$CERTBOT_DOMAIN" = "$ROOT_DOMAIN" ]; then
    RECORD_NAME="_acme-challenge"
else
    SUBDOMAIN=$(echo "$CERTBOT_DOMAIN" | sed "s/\.${ROOT_DOMAIN}$//")
    RECORD_NAME="_acme-challenge.${SUBDOMAIN}"
fi

echo "Setting DNS TXT record: ${RECORD_NAME}.${ROOT_DOMAIN} = ${CERTBOT_VALIDATION}"

# Add TXT record via GoDaddy API
curl -s -X PATCH \
    "https://api.godaddy.com/v1/domains/${ROOT_DOMAIN}/records" \
    -H "Authorization: sso-key ${GODADDY_API_KEY}:${GODADDY_API_SECRET}" \
    -H "Content-Type: application/json" \
    -d "[{\"type\":\"TXT\",\"name\":\"${RECORD_NAME}\",\"data\":\"${CERTBOT_VALIDATION}\",\"ttl\":600}]"

# Wait for DNS propagation
echo "Waiting 30 seconds for DNS propagation..."
sleep 30
AUTHEOF

chmod +x "${CERTBOT_PATH}/godaddy-auth.sh"

# Create GoDaddy DNS cleanup script
cat > "${CERTBOT_PATH}/godaddy-cleanup.sh" << 'CLEANEOF'
#!/bin/bash
# GoDaddy DNS Cleanup for Certbot

GODADDY_API_KEY="$GODADDY_API_KEY"
GODADDY_API_SECRET="$GODADDY_API_SECRET"

CERTBOT_DOMAIN="${CERTBOT_DOMAIN}"

# For wildcard or subdomain, we need the root domain
ROOT_DOMAIN=$(echo "$CERTBOT_DOMAIN" | awk -F. '{print $(NF-1)"."$NF}')

# Determine the record name
if [ "$CERTBOT_DOMAIN" = "$ROOT_DOMAIN" ]; then
    RECORD_NAME="_acme-challenge"
else
    SUBDOMAIN=$(echo "$CERTBOT_DOMAIN" | sed "s/\.${ROOT_DOMAIN}$//")
    RECORD_NAME="_acme-challenge.${SUBDOMAIN}"
fi

echo "Cleaning up DNS TXT record: ${RECORD_NAME}.${ROOT_DOMAIN}"

# Delete TXT record via GoDaddy API
curl -s -X DELETE \
    "https://api.godaddy.com/v1/domains/${ROOT_DOMAIN}/records/TXT/${RECORD_NAME}" \
    -H "Authorization: sso-key ${GODADDY_API_KEY}:${GODADDY_API_SECRET}"
CLEANEOF

chmod +x "${CERTBOT_PATH}/godaddy-cleanup.sh"

# Build staging argument
STAGING_ARG=""
if [ "$STAGING" = "1" ]; then
    STAGING_ARG="--staging"
    echo "WARNING: Using Let's Encrypt staging environment (for testing)"
fi

# Request certificate using DNS challenge
echo ""
echo "Requesting certificate for ${DOMAIN} and www.${DOMAIN}..."
echo ""

# Export variables for the auth scripts
export GODADDY_API_KEY
export GODADDY_API_SECRET

docker run --rm \
    -v "${CERTBOT_PATH}/conf:/etc/letsencrypt" \
    -v "${CERTBOT_PATH}/logs:/var/log/letsencrypt" \
    -v "${CERTBOT_PATH}:/certbot-scripts" \
    -e GODADDY_API_KEY="${GODADDY_API_KEY}" \
    -e GODADDY_API_SECRET="${GODADDY_API_SECRET}" \
    certbot/certbot certonly \
    --manual \
    --preferred-challenges dns \
    --manual-auth-hook "/certbot-scripts/godaddy-auth.sh" \
    --manual-cleanup-hook "/certbot-scripts/godaddy-cleanup.sh" \
    --email "${EMAIL}" \
    --agree-tos \
    --no-eff-email \
    ${STAGING_ARG} \
    -d "${DOMAIN}" \
    -d "www.${DOMAIN}"

echo ""
echo "=== SSL Certificate Setup Complete ==="
echo "Certificate obtained for: ${DOMAIN}"
echo ""
echo "Certificate files:"
echo "  - ${CERTBOT_PATH}/conf/live/${DOMAIN}/fullchain.pem"
echo "  - ${CERTBOT_PATH}/conf/live/${DOMAIN}/privkey.pem"
echo ""
echo "Now you can start the services with SSL:"
echo "  cd ${PROJECT_DIR}"
echo "  docker compose -f docker-compose.prod.yml --env-file deploy/.env up -d"
echo ""
echo "To renew certificates, run:"
echo "  GODADDY_API_KEY=xxx GODADDY_API_SECRET=xxx ./scripts/renew-cert.sh"
