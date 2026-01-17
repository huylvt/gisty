# Gisty 🚀

**Gisty** là một nền tảng mã nguồn mở cho phép chia sẻ các đoạn mã (snippets) và văn bản thô với tốc độ cực nhanh, giao diện tối giản và khả năng mở rộng mạnh mẽ.

---

## 📖 Tài liệu dự án

Để hiểu rõ về tầm nhìn, tính năng và lộ trình phát triển của hệ thống, vui lòng tham khảo tài liệu đặc tả sản phẩm:

👉 **[Tài liệu Yêu cầu Sản phẩm (PRD)](./docs/PRD.md)**

---

## ✨ Tính năng nổi bật

* **Siêu nhanh:** Tối ưu hóa độ trễ cho việc tạo và đọc paste.
* **Syntax Highlighting:** Hỗ trợ tô sáng mã nguồn tự động.
* **Tùy chỉnh thời hạn:** Hỗ trợ tính năng "Xem xong xóa" hoặc đặt thời gian hết hạn linh hoạt.
* **API-First:** Dễ dàng tích hợp vào các công cụ dòng lệnh (CLI) hoặc ứng dụng khác.

## 🏗️ Kiến trúc tổng quan

Dự án được xây dựng dựa trên các trụ cột kỹ thuật:
* **Backend:** Go.
* **Database:** MongoDB (Metadata) & Redis (Cache).
* **Storage:** Amazon S3 hoặc MinIO (Content).
* **ID Generation:** Key Generation Service (KGS) dựa trên Base62.

## 🛠️ Cấu trúc thư mục

```text
Gisty/
├── docs/
│   └── PRD.md            # Tài liệu yêu cầu sản phẩm chi tiết
├── src/                  # Mã nguồn ứng dụng
├── public/               # Tài nguyên tĩnh (CSS, JS, Images)
├── docker-compose.yml    # Cấu hình môi trường Docker
└── README.md             # Tài liệu hướng dẫn này
```

## 🚀 Bắt đầu nhanh (Local Setup)

1. Clone repo:
    ```bash
    git clone [https://github.com/huylvt/gisty.git](https://github.com/huylvt/gisty.git)
    cd gisty
    ```

2. Khởi chạy với Docker:
    ```bash
    docker-compose up -d
    ```

3. Truy cập hệ thống tại: http://localhost:3000
