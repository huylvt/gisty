# Deployment Guide: Gisty.co

## Prerequisites

- Ubuntu 22.04 LTS or higher
- Docker & Docker Compose installed
- Domain name pointing to your server
- GitHub repository with CI/CD configured

## GitHub Secrets Configuration

Add these secrets to your GitHub repository (Settings > Secrets and variables > Actions):

### Required Secrets

| Secret | Description | Example |
|--------|-------------|---------|
| `SSH_HOST` | Server IP or hostname | `203.0.113.10` |
| `SSH_USERNAME` | SSH username | `huylvt` |
| `SSH_PRIVATE_KEY` | SSH private key | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `SSH_PORT` | SSH port (optional) | `22` |
| `APP_URL` | Public URL | `https://gisty.co` |
| `MONGO_URI` | MongoDB connection string | `mongodb://user:pass@host:27017/db` |
| `MONGO_DB` | Database name | `gisty` |
| `MONGO_USERNAME` | MongoDB username | `gisty` |
| `MONGO_PASSWORD` | MongoDB password | `secure-password` |
| `REDIS_URI` | Redis connection string | `redis://localhost:6379` |
| `S3_ACCESS_KEY_ID` | S3/MinIO access key | `AKIAXXXXXXXX` |
| `S3_SECRET_ACCESS_KEY` | S3/MinIO secret key | `wJalrXXXXXXXX` |
| `S3_BUCKET_NAME` | S3 bucket name | `gisty` |
| `S3_REGION` | S3 region | `us-east-1` |
| `S3_ENDPOINT` | S3 endpoint (for S3-compatible) | `https://s3.amazonaws.com` |
| `DOCKERHUB_USERNAME` | Docker Hub username | `huylvt` |
| `DOCKERHUB_TOKEN` | Docker Hub access token | `dckr_pat_xxxxx` |
| `API_URL` | API URL for frontend build | `https://gisty.co` |

## Initial Server Setup

### 1. Setup User

```bash
# On your server, ensure huylvt user has docker access
sudo usermod -aG docker huylvt
```

### 2. Setup SSH Key Authentication

```bash
# On your local machine
ssh-keygen -t ed25519 -C "huylvt@gisty.co"

# Copy public key to server
ssh-copy-id -i ~/.ssh/id_ed25519.pub huylvt@your-server

# Add private key to GitHub Secrets as SSH_PRIVATE_KEY
```

### 3. Create Application Directory

```bash
# On your server
sudo mkdir -p /srv/gisty
sudo chown huylvt:huylvt /srv/gisty
```

### 4. Initial Deployment

```bash
# Clone repository
cd /srv/gisty
git clone https://github.com/huylvt/gisty.git .

# Create directories
mkdir -p certbot/conf certbot/www

# Copy environment file
cp .env.example deploy/.env
# Edit deploy/.env with your production values

# Initialize SSL certificates
chmod +x scripts/init-letsencrypt.sh
DOMAIN=gisty.co EMAIL=admin@gisty.co ./scripts/init-letsencrypt.sh

# Start all services
docker compose -f docker-compose.prod.yml --env-file deploy/.env up -d
```

## Deployment Flow

After initial setup, deployments happen automatically via GitHub Actions:

1. Push code to `main` branch
2. CI pipeline runs tests
3. Docker images are built and pushed to Docker Hub
4. CD pipeline deploys to server via SSH
5. Health check verifies deployment

## Manual Deployment

If you need to deploy manually:

```bash
# SSH into server
ssh huylvt@your-server

cd /srv/gisty

# Pull latest images
docker compose -f docker-compose.prod.yml --env-file deploy/.env pull

# Deploy with zero downtime
docker compose -f docker-compose.prod.yml --env-file deploy/.env up -d --remove-orphans

# Verify deployment
curl http://localhost:8080/health
```

## Monitoring & Logs

### View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f gisty

# Last 100 lines
docker compose -f docker-compose.prod.yml logs --tail=100 gisty
```

### Health Checks

```bash
# Backend health
curl https://gisty.co/health

# Check container status
docker compose -f docker-compose.prod.yml ps
```

## Backup & Recovery

### MongoDB Backup

```bash
# Manual backup
docker exec gisty-mongodb /scripts/backup-mongodb.sh

# Setup cron job (runs daily at 2 AM)
crontab -e
# Add: 0 2 * * * docker exec gisty-mongodb /scripts/backup-mongodb.sh
```

### Restore from Backup

```bash
# Download backup from S3
aws s3 cp s3://gisty-backups/mongodb/gisty_20240115_020000.tar.gz /tmp/

# Extract
cd /tmp && tar -xzf gisty_20240115_020000.tar.gz

# Restore
docker exec -i gisty-mongodb mongorestore --drop /tmp/gisty_20240115_020000/
```

## SSL Certificate Renewal

Certificates are renewed automatically by Certbot. To manually renew:

```bash
docker compose -f docker-compose.prod.yml run --rm certbot renew
docker compose -f docker-compose.prod.yml exec nginx nginx -s reload
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker compose -f docker-compose.prod.yml logs gisty

# Check environment variables
docker compose -f docker-compose.prod.yml config
```

### Database Connection Issues

```bash
# Test MongoDB connection
docker exec gisty-mongodb mongosh -u gisty -p --eval "db.adminCommand('ping')"

# Check network connectivity
docker network inspect gisty_gisty-network
```

### Nginx Issues

```bash
# Test nginx configuration
docker exec gisty-nginx nginx -t

# Reload nginx
docker exec gisty-nginx nginx -s reload
```

## Rollback

To rollback to a previous version:

```bash
# Find previous image tag
docker images huylvt/gisty

# Update deploy/.env with previous tag
BACKEND_IMAGE=huylvt/gisty:previous-sha
FRONTEND_IMAGE=huylvt/gisty-web:previous-sha

# Redeploy
docker compose -f docker-compose.prod.yml --env-file deploy/.env up -d
```
