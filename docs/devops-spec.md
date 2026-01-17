# 🛠️ DevOps Specification: Gisty.co

Tài liệu này quy định các tiêu chuẩn về vận hành, đóng gói và triển khai hệ thống Gisty.co.

## 1. Chiến lược Containerization (Docker)

Mọi thành phần của hệ thống phải được chạy trong Docker container để đảm bảo tính nhất quán giữa các môi trường (Dev, Staging, Production).

### 1.1. Dockerfile Standards
- Base Image: Sử dụng bản alpine hoặc slim (ví dụ: node:20-alpine) để giảm thiểu dung lượng và lỗ hổng bảo mật.

- Multi-stage Build: Bắt buộc sử dụng để tách biệt môi trường build và môi trường chạy.

- User: Không chạy ứng dụng bằng quyền root. Sử dụng USER node hoặc tạo user mới.

- Layer Cache: Sắp xếp COPY package*.json hoặc COPY go.mod trước khi COPY toàn bộ source code để tận dụng cache.

### 1.2. Orchestration
- Development: Sử dụng docker-compose.yml để chạy toàn bộ stack cục bộ (App, DB, Redis, MinIO).

- Production: Sử dụng docker-compose.prod.yml.

## 2. Quy trình CI/CD (GitHub Actions)
### 2.1. Pipeline CI (Continuous Integration)
- Chạy tự động khi có Pull Request hoặc Push vào nhánh develop/main.

- Linting & Security: Kiểm tra lỗi cú pháp và quét lỗ hổng phụ thuộc (npm audit/snyk).

- Unit Testing: Chạy bộ test suite của Backend và Frontend.

- Build Image: Build Docker image và tag theo mã hash của Git commit (sha-xxxx).

### 2.2. Pipeline CD (Continuous Deployment)
- Chạy tự động khi code được merge vào nhánh main.

- Registry: Push image lên Docker Hub hoặc GitHub Packages (GHCR).

- SSH Deployment:

  - Kết nối tới VPS qua SSH.

  - Cập nhật file .env từ GitHub Secrets.

  - Chạy docker compose pull và docker compose up -d --remove-orphans.

  - Kiểm tra Health Check (trang web phải trả về status 200).

## 3. Quản lý cấu hình & Bí mật (Secrets)
Các biến nhạy cảm không được lưu trong code hoặc Git.

| Biến | Nguồn | Ghi chú |
|--|--|--|
|S3_ACCESS_KEY | GitHub Secrets | Dùng cho kết nối Storage |
|MONGO_URL | GitHub Secrets | Chuỗi kết nối Database |

## 4. Hạ tầng Production (Infrastructure)
- Server: Ubuntu 22.04 LTS hoặc cao hơn.

- Storage: S3-compatible.

## 5. Chiến lược Sao lưu (Backup)
- Database: Backup MongoDB định kỳ hàng ngày (Mongodump) và đẩy lên một S3 bucket riêng (Private).

- Code: Toàn bộ cấu hình hạ tầng (Nginx config, Dockerfiles) phải nằm trong Git.
