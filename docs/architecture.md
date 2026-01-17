# 🏗️ System Architecture: Gisty
Tài liệu này mô tả chi tiết kiến trúc kỹ thuật, luồng dữ liệu và các quyết định thiết kế cho hệ thống Gisty.

## 1. Thành phần hệ thống (High-Level Components)
Hệ thống được thiết kế theo kiến trúc Microservices (hoặc Modular Monolith cho giai đoạn đầu) bao gồm:
- API Gateway / Load Balancer: Sử dụng Nginx để phân phối lưu lượng và SSL termination.
- Write Service: Xử lý việc tạo mới các bản Gisty, nén dữ liệu và lưu trữ.
- Read Service: Truy xuất dữ liệu từ Cache hoặc Storage. Tối ưu hóa cho tốc độ đọc.
- Key Generation Service (KGS): Một dịch vụ độc lập chuyên tạo trước các mã định danh (Unique IDs) không trùng lặp.
- Cleanup Worker: Dịch vụ chạy ngầm để xóa các bản ghi đã hết hạn (TTL).

## 2. Mô hình dữ liệu (Data Models)
### 2.1. Metadata Database (NoSQL - MongoDB/Cassandra)
Lưu trữ thông tin quản lý của mỗi bản Gisty.

```json
{
  "short_id": "xK9a2",          // Primary Key (Base62)
  "user_id": "uuid-string",     // Optional
  "content_key": "s3-link-path",// Đường dẫn tới file vật lý
  "expiration_date": "timestamp",
  "created_at": "timestamp",
  "syntax_type": "python",
  "is_private": false
}
```
## 2.2. Object Storage (S3 / MinIO)
- Hệ thống sử dụng Amazon S3 (hoặc tương đương) làm lưu trữ chính.
- Dữ liệu văn bản sẽ được lưu dưới dạng file .txt hoặc .bin với tên file là short_id.
- Các object sẽ được lưu trong bucket và có prefix là `/gisty`
- Để tối ưu, các file này sẽ được thiết lập Header Content-Type: text/plain và sử dụng cơ chế S3 Lifecycle Policy để tự động xóa các file hết hạn (nếu cần).

## 3. Luồng dữ liệu (Data Flow)
### 3.1. Quy trình Ghi (Write Path)
- User gửi nội dung tới POST /api/v1/pastes.
- App Server lấy một short_id duy nhất từ KGS.
- Nội dung văn bản được đẩy vào Object Storage.
- Metadata (bao gồm cả link trỏ tới Object Storage) được lưu vào NoSQL DB.
- Đẩy dữ liệu vào Redis Cache (nếu cần thiết).
- Trả về URL: gisty.io/{short_id}.

### 3.2. Quy trình Đọc (Read Path)
- User truy cập GET /gisty.io/{short_id}.
- Server kiểm tra trong Redis Cache.
  - Nếu có (Cache Hit): Trả về ngay lập tức.
  - Nếu không (Cache Miss): Truy vấn NoSQL DB để lấy Metadata.
- Nếu tìm thấy Metadata, lấy nội dung từ Object Storage.
- Cập nhật nội dung vào Cache và trả về cho User.

## 4. Key Generation Service (KGS)
Để tránh xung đột ID khi hệ thống mở rộng (horizontal scaling), KGS sẽ:
- Sinh trước hàng triệu mã Base62 (ví dụ: a7B2k9).
- Lưu vào hai bảng: key_used và key_unused.
- Khi App Server cần ID, KGS chỉ việc bốc một mã từ key_unused và chuyển sang key_used.
- Điều này giúp việc tạo ID có độ trễ gần như bằng 0 ($O(1)$).

## 5. Chiến lược Caching & Tối ưu
- Lru Cache (Least Recently Used): Chỉ giữ những bản Gisty "hot" nhất trong RAM.
- Content Compression: Sử dụng Gzip hoặc Zstd để nén văn bản trước khi lưu vào Storage (giảm ~50% dung lượng).
- CDN (Content Delivery Network): Sử dụng Cloudflare hoặc CloudFront để cache các bản Gisty công khai ở các node gần người dùng nhất.

### 6. Sơ đồ kiến trúc (Dạng Text)
```text
User -> Load Balancer -> App Servers
                          |
        +-----------------+-----------------+
        |                 |                 |
  [ KGS Service ]   [ NoSQL DB ]     [ Object Storage ]
        |                 |                 |
  (Pre-gen IDs)     (Metadata)       (Raw Content)
                          |
                    [ Redis Cache ]
```
