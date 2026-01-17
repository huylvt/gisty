# 🎨 Frontend Specification: Gisty.co

## 1. Công nghệ (Tech Stack)
- Framework: React.js (để tối ưu SEO và tốc độ tải trang).

- Styling: Tailwind CSS (để tùy chỉnh giao diện nhanh theo bộ nhận diện thương hiệu).

- Icons: Lucide React.

- Code Editor Component: Monaco Editor.

## 2. Giao diện người dùng (UI Components)
### 2.1. Thanh điều hướng (Navbar)
- Trái: Logo Gisty.co (Sử dụng logo đã thiết kế) trỏ về trang chủ.

- Phải: Nút "New Paste", "Login/Sign up" (Ghost button), và Toggle "Dark/Light Mode".

- Đặc điểm: Cố định (sticky) khi cuộn trang, nền kính mờ (glassmorphism).

### 2.2. Trang chủ / Trình soạn thảo (Home/Editor)
- Vùng nhập liệu (Main Editor): Chiếm 70-80% chiều cao màn hình. Hỗ trợ tự động giãn nở (auto-resize) hoặc scroll trong khung cố định.

- Thanh công cụ bên dưới/bên cạnh:

  - Dropdown chọn ngôn ngữ (Default: Auto-detect).

  - Dropdown chọn thời gian hết hạn (Expiration).

- Nút "Save/Gistify" (Màu xanh Primary, nổi bật nhất).

### 2.3. Trang hiển thị (View Paste)
- Header: Hiển thị tên file (nếu có), ngôn ngữ, thời gian đã tạo và lượt xem.

- Nút chức năng: "Copy Raw", "Download", "Clone/Edit" (tạo bản copy mới).

- Vùng hiển thị: Code được render với Syntax Highlighting chuẩn, có số thứ tự dòng (line numbers).

## 3. Trải nghiệm người dùng (UX & Interactions)
- Phím tắt (Hotkeys): Ctrl + S hoặc Cmd + S để lưu Paste ngay lập tức.

- Thông báo (Toast Notifications): Hiển thị ở góc trên bên phải khi:

  - Copy link thành công.

  - Paste thành công (trả về URL).

  - Lỗi khi file quá lớn (>10MB).

- Trạng thái tải (Loading): Sử dụng Progress bar chạy trên đỉnh trang khi đang upload nội dung lên S3.

## 4. Đặc tả màu sắc & Font chữ (CSS Variables)
Yêu cầu Claude khởi tạo file globals.css với các biến sau:

```css
:root {
  --primary: #00D1FF;     /* Màu xanh chủ đạo */
  --bg-dark: #0F172A;     /* Màu nền Navy đậm */
  --editor-bg: #1E293B;   /* Màu nền khung code */
  --text-main: #F8FAFC;   /* Màu chữ chính */
  --accent: #6366F1;      /* Màu tím indigo bổ trợ */
}
```

## 5. Cấu trúc trang (Pages)
- / : Trang chủ, chứa trình soạn thảo trống.

- /:id : Trang xem nội dung đoạn mã đã lưu.

- /u/:username : (Giai đoạn 2) Danh sách các paste của người dùng.