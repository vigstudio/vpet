# Thú Ảo — Starter UI Kit

Thư viện **starter** theo phong cách fantasy/chibi 2.5D. Bao gồm 7 sheet PNG RGBA nền trong suốt, 5 bản thiết kế tham khảo, CSS component responsive, và `index.html` demo mở trực tiếp bằng trình duyệt.

## Các thư mục
- `assets/sheets`: sprite sheets (UI, icon, pet, buildings, map, items, skills).
- `assets/screens`: ảnh mockup tham khảo, không phải UI triển khai.
- `css/game-ui.css`: button (default/green/blue/purple/disabled), panel, tab, progress, quest, equipment slot, popup, chat bubble.
- `index.html`: demo với tương tác popup/tab.
- `manifest.json`: danh sách tài nguyên.

## Cách dùng
Mở `index.html`, hoặc copy `css/game-ui.css` và `assets` vào dự án. Dùng CSS/HTML thật cho văn bản, số liệu, thanh tiến độ và button; không dựng UI bằng ảnh mockup toàn màn hình.

## Lưu ý
Các hình PNG được AI tạo dưới dạng **sprite sheet tổng hợp**; từng vật thể chưa được cắt/kiểm duyệt pixel-perfect. Để dùng trong production, cần tách riêng icon/character thành texture với bounding box, kiểm tra viền alpha, đồng nhất tỉ lệ, dùng atlas tối ưu WebP/PNG. Tránh dùng chữ được vẽ sẵn trong ảnh để hiển thị nội dung động. Các nhân vật, icon và bối cảnh minh họa cần kiểm tra chất lượng, cách đặt tên và kỹ thuật animation trước khi tích hợp chính thức.
