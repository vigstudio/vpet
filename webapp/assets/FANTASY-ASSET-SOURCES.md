# Bộ fantasy do người dùng cung cấp

Ngày nhận: 31/10/2026. Người dùng cho biết đã nhờ ChatGPT tạo và yêu cầu cắt để dùng trong VPet. Đây là ghi nhận nguồn cung cấp, không phải chứng nhận giấy phép độc quyền hoặc CC0.

## Nguồn lưu trong repo

`fantasy-source/` giữ bảy ảnh gốc, mỗi ảnh 1448×1086:

- `ui.png`: ảnh #12, nút, khung, họa tiết.
- `icons.png`: ảnh #13, điều hướng, chức năng, avatar trang trí.
- `14.png`: kỹ năng, trạng thái, hiệu ứng.
- `15.png`: thức ăn, vật phẩm, rương, trang bị.
- `16.png`: công trình.
- `17.png`: nhân vật, quái, biểu cảm.
- `18.png`: cây, đường, cầu, đảo, họa tiết môi trường.

Ảnh #19 trùng byte với #15; #20 trùng byte với #14. Không lưu thêm bản trùng.

## Kết quả

208 PNG RGBA trong `fantasy/`; `manifest.json` ghi SHA-256 nguồn, tọa độ crop, kích thước đầu ra. `preview.jpg` là contact sheet kiểm tra, không dùng trong runtime.

Chạy lại từ gốc repo, với Python có Pillow:

```bash
python webapp/assets/cut-fantasy-assets.py \
  webapp/assets/ui-kit-source/ui.png \
  webapp/assets/ui-kit-source/icons.png \
  --extras-dir webapp/assets/ui-kit-source \
  --source-dir webapp/assets/ui-kit-source
```

Script kiểm tra kích thước ảnh nguồn, ảnh đầu ra đọc được và alpha vừa có nền trong suốt vừa có artwork đục. Đây không phải kiểm thử tích hợp giao diện.

## Xử lý và giới hạn

- Ảnh gốc có nền đen đục. Chỉ loại pixel gần đen nối tới biên crop; giữ phần nét đen nằm kín bên trong artwork.
- Glow và bóng được vẽ sẵn trên nền đen có thể còn viền tối; phải xem trên nền sử dụng trước khi đưa vào game. Tách nền không khôi phục được alpha nguyên bản của ảnh đã flatten.
- 11 icon điều hướng bỏ biển chữ, giữ nhãn HTML để dịch vi/en/ja.
- Sáu nút dùng dải giữa không chữ để bỏ dấu chân; không lấy chữ/số/tiến độ mẫu làm dữ liệu runtime.
- Chỉ cắt phần dùng được. Không nhập các dialog, thanh tiền hoặc meter có chữ/số mẫu thành UI giả.
- Các crop chỉ là bản tách ban đầu; kiểm tra rìa, thành phần lân cận và kích thước ở từng vị trí dùng thực tế.
- Nhân vật minh họa không thay pet cá nhân được dựng từ `parts`. Quái/kỹ năng chỉ gắn theo ID và luật hiện có, không tự thêm cơ chế game.
- React dùng icon điều hướng, chín công trình, icon nhiệm vụ, nút vàng và khung gỗ. Webpack xuất file riêng có hash; không nhúng cả sheet hay nguồn vào JS.
- `make-fantasy-scenes.py` ghép Thành phố 1600×900 và sân chăm thú 1000×700 từ artwork được cung cấp. Cảnh ghép không phải tranh toàn cảnh gốc; không khẳng định khớp ảnh tham chiếu từng pixel.

## ZIP starter và ảnh #21 bổ sung

- `thu-ao-ui-kit-starter.zip` do người dùng gửi cùng ngày: bảy sheet RGBA có alpha, năm mockup và HTML/CSS demo. `ui-kit-source/` giữ bảy sheet cùng README/manifest nguồn; manifest tự ghi `project-use`, không phải giấy phép CC0 hay quyền độc quyền đã xác minh.
- `fantasy/` hiện tái cắt từ bản alpha trong ZIP, giữ alpha sẵn thay vì flood-fill đen. Nguồn ảnh đen cũ vẫn giữ riêng để đối chiếu.
- `fantasy-source/21.png` là bảng tổng hợp 1536×1024 mới. `cut-library-assets.py` cắt 98 PNG vào `library/`, gồm icon, công trình, đồ, nền cảnh và panel. Manifest ghi tọa độ và SHA-256; preview trên nền giấy dùng để kiểm tra crop.
- Nền tím ảnh #21 được loại ở vùng nối biên. Một số pixel tối/viền hoặc hiệu ứng có thể chưa hoàn hảo; scene chỉ là thumbnail nhỏ, không phải tranh HD. Không lấy chữ/số mẫu làm dữ liệu game; không thay `PetCanvas` bằng pet mẫu.
- Chạy lại: `python webapp/assets/cut-library-assets.py webapp/assets/fantasy-source/21.png`, rồi `python webapp/assets/make-fantasy-scenes.py`.
- Smoke local: Thành phố/Nhiệm vụ/Thú cưng ở 1440/1024/375, CTA chiến dịch ở 740×375 và 844×390; không pageerror/tràn ngang, ảnh tải đủ. Kiểm thử này không xác nhận mọi crop hoặc chất lượng tranh HD.
