# Product Requirements Document: Gisty


Thuộc tính	Chi tiết
Sản phẩm	Gisty (Snippet Sharing Platform)
Phiên bản	1.0.0
Trạng thái	Draft (Bản thảo)
Ngày khởi tạo	14/01/2026
Chủ sở hữu	Phát triển bởi Gemini Thought Partner

---

## 1. Tầm nhìn sản phẩm (Product Vision)
Gisty được định vị là một nền tảng lưu trữ văn bản và mã nguồn siêu nhanh, tối giản. Không quảng cáo gây phiền nhiễu, không rườm rà, tập trung tối đa vào trải nghiệm "Paste & Share" dành cho cộng đồng công nghệ.

## 2. Mục tiêu chiến lược

- Tốc độ: Thời gian từ lúc dán đến lúc có link chia sẻ < 2 giây.

- Độ tin cậy: Hệ thống lưu trữ phân tán đảm bảo dữ liệu không bị mất mát.

- Sự đơn giản: Người dùng không cần tài khoản vẫn có thể sử dụng đầy đủ tính năng cốt lõi.

## 3. Phân tích tính năng (Features)
### 3.1. Nhóm tính năng MVP (Giai đoạn 1)
- Quick Paste: Ô nhập liệu hỗ trợ Plain Text và Code.

- Automatic Syntax Highlighting: Tự động phát hiện ngôn ngữ lập trình dựa trên nội dung.

- Short Link Generation: Tự động tạo URL rút gọn bằng Base62 (ví dụ: gisty.io/xK9a2).

- Expiration Settings:

  - Burn on read (Xem xong xóa ngay).

  - Timed (10 phút, 1 giờ, 1 ngày, 1 tuần).

  - Never (Vĩnh viễn).

- One-Click Copy: Sao chép nội dung hoặc URL chỉ với một lần bấm.

### 3.2. Nhóm tính năng Nâng cao (Giai đoạn 2)
- Private/Unlisted Pastes: Ẩn paste khỏi công cụ tìm kiếm và danh sách công khai.

- Password Protection: Yêu cầu mật khẩu để xem nội dung.

- User Dashboard: Quản lý, chỉnh sửa hoặc xóa các bản Gisty đã tạo.

- API for Developers: Cung cấp token để tạo paste qua Terminal hoặc ứng dụng bên thứ ba.

## 4. Yêu cầu kỹ thuật & Kiến trúc (Technical Specs)
### 4.1. Hệ thống lưu trữ
- Metadata (Database): NoSQL (MongoDB/Cassandra) để lưu trữ thông tin điều hướng.

- Content (Object Storage): Lưu nội dung vào các File thô để tối ưu chi phí và dung lượng.

- Caching Strategy: Sử dụng Redis cho các "Hot Gisties" (các bản ghi có lượt xem cao trong thời gian ngắn).

### 4.2. Bảo mật & Chống Spam
- Rate Limiting: Giới hạn 5 request/phút cho mỗi địa chỉ IP để tránh spam bot.

- Sanitization: Làm sạch nội dung văn bản để tránh các cuộc tấn công XSS (Cross-Site Scripting).

### 5. Thiết kế giao diện (UI/UX)
- Phong cách: Minimalist (Tối giản), tập trung vào Typography.

- Theme: Mặc định là Dark Mode (có tùy chọn chuyển sang Light Mode).

- Editor: Sử dụng thư viện như CodeMirror hoặc Monaco Editor để có trải nghiệm giống VS Code.

### 6. Lộ trình phát triển (Roadmap)
- Phase 1: Hoàn thiện Backend Core (KGS, Storage Service, API).

- Phase 2: Hoàn thiện Frontend MVP và tích hợp Syntax Highlighting.

- Phase 3: Thử nghiệm Beta và tối ưu hóa hệ thống Cache.

- Phase 4: Ra mắt chính thức và giới thiệu API Public.

### 7. Chỉ số thành công (KPIs)
- Uptime: Duy trì mức 99.9%.

- User Retention: Tỷ lệ người dùng quay lại sử dụng sau lần đầu.

- Load Time: Thời gian tải trang dưới 500ms cho các nội dung đã được cache.