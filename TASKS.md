# VPet — checklist phát hành 31/10/2026

Đã viết ≠ đã kiểm thử ≠ đã deploy. Giữ mọi task bên dưới khi đổi hướng mỹ thuật. Chỉ đóng sau kiểm thử; không đưa secret, dữ liệu người chơi hoặc thông tin vận hành vào đây.

## Yêu cầu đã duyệt

- UI fantasy sáng: ảnh Thành phố làm chuẩn đảo bay/thanh gỗ/chat giấy; ảnh Nhiệm vụ làm chuẩn nội dung giấy, tiến độ và thưởng.
- Thành phố đủ 9 công trình đúng đích; 11 tab desktop, mobile 4 tab + Thêm. Không biến game trở lại modal.
- Chữ/nút là DOM thật, vi/en/ja, focus rõ, target ≥44px, reduced-motion, cảnh 16:9, không tràn ngang 375px.
- Ảnh tham chiếu không phải asset sạch. Không chép tên/chat/số dư trong ảnh vào game. Không thêm nạp tiền hay tính năng mới chỉ vì icon trong ảnh.
- Giữ gameplay cũ, an toàn giao dịch, replay/cursor SSE, tài khoản và dữ liệu hiện có.

## Tiến độ

| Hạng mục | Trạng thái | Điều kiện còn thiếu |
|---|---|---|
| Alliance / lãnh thổ / thưởng mùa | Backend nền đã phát hành; UI mới đã viết | Mỹ thuật, E2E quyền/đánh/settlement/claim |
| Săn đội | UI đội hình đã viết | Mỹ thuật, tạo/vào/xuất phát/nhận quà |
| Cửa hàng | Quầy/card đã viết | Kiểm tra mua/stock và responsive |
| Gacha | Reel 5,12 giây + helper/test đã viết | Winner dưới kim, trừ tiền một lần, unmount, reduced-motion |
| Chiến dịch | 12 ải/vùng, 60 ải; code/test đã viết | Browser hai chặng, sao cũ, khóa/mở, i18n |
| Leo tháp | 50 tầng, 10 lượt/ngày; backend/UI/test đã viết | Browser/persistence, thông báo đủ sách/ngọc/skill, không thưởng lặp |
| Giấy đổi skill | Chọn skill cũ/mới; giữ cấp và skill khác | E2E, request lỗi không mất sách |
| Tối Thượng / Hồng Mông | Rarity 6 đã viết; chưa kiểm tra độc lập toàn diff | 4 divine + 5000 vàng + 3 Tinh Hồn; forge/stats/nguồn thưởng |
| Tinh Hồn | Drop boss/đất tier 3, khóa giao dịch đã viết | War drop, boss settlement/claim idempotence |
| Nhạc nền | Có sẵn | Regression toggle/nhớ lựa chọn/ẩn tab |
| Replay tự hiện thưởng | Có sẵn và tích hợp tháp | Callback một lần, pause/skip/reduced-motion |
| Landscape | Đã sửa, CTA trong viewport 740×375 và 844×390 qua Playwright | Nhãn ải trên cảnh nhỏ còn dày, cần tinh chỉnh mỹ thuật |
| Bộ asset fantasy | ZIP starter: tái cắt 208 PNG giữ alpha; ảnh #21: thêm 98 PNG và manifest; đã ghép hai cảnh WebP | Nguồn/giới hạn tại `webapp/assets/FANTASY-ASSET-SOURCES.md`; một số crop/FX cần kiểm duyệt thêm, thumbnail cảnh không phải tranh HD |
| Vỏ / Thành phố / Nhiệm vụ | Đã tích hợp 11 icon tranh, 9 công trình DOM, khung gỗ, sân pet; nhiệm vụ ba cột ở desktop rộng | Smoke local 1440/1024/375 không pageerror/tràn ngang; cảnh ghép chưa tương đương tranh toàn cảnh reference, tiếp tục tinh chỉnh mỹ thuật |
| Chat / hồ sơ | Đổi khung giấy, giữ nguyên logic | Tên/tin dài, keyboard mobile, reconnect chưa regression đầy đủ |
| Go / frontend | Go toàn suite xanh; frontend 78/78 xanh; webpack và Docker build thành công | Store DB integration không bật, tự skip; JS khoảng 729 KiB, artwork xuất file riêng; còn cảnh báo kích thước |
| Browser smoke | Desktop 1024/mobile 375: Alliance, săn đội, tháp, nhiệm vụ, pet, thành phố, shop xanh; gacha và đổi skill xanh | CTA ngang đã sửa và assert xanh ở 740×375, 844×390; tổng 18 kiểm tra smoke xanh |
| Git / deploy đợt này | Artwork `4fac173` đã push và deploy; app/DB healthy; backup DB đã kiểm tra bằng pg_restore, có image rollback riêng | Smoke local 9 tổ hợp và 2 landscape đạt; mỹ thuật cảnh HD vẫn chưa hoàn tất; kiểm tra public ghi bên dưới |

## Chợ / Thú cưng theo ảnh #24–#25

- Đã viết: Chợ hai quầy giấy/gỗ, mái bạt đỏ kem, icon vàng, kho chọn món, số lượng/tổng giá lô và CTA xanh. Giữ bán/cần mua, rút tin, trao đổi và luật phí 5 vàng/thuế 10% từ server.
- Đã viết: một selector pet chuyển ngang/dọc; cảnh pet thật với tám slot đồ và bệ đứng, chăm sóc/khay đồ, skill có tranh theo ID, bộ đồ/XP chuyển xuống dưới. Một pet kín slot không bị bó vào cột sidebar; không pet không tạo selector rỗng.
- Đã kiểm thử local: chợ hai tài khoản đăng/mua/rút, fulfill/cancel đơn cần mua; phí, thuế, vàng/kho đúng, mua lại tin đã bán bị từ chối. Dùng đồ/nâng skill/mặc-tháo gear đúng pet thứ hai.
- Render: 1672/1440/1024/375, không tràn ngang hoặc pageerror; pet chết/nghỉ dưỡng/kho trống/no-pet/một slot, locale en/ja đạt smoke. Đã xem ảnh chợ có hàng và pet; nền pet vẫn từ thumbnail, chưa phải tranh HD như mẫu.
- Landscape 740×375, 844×390: CTA truy cập được khi cuộn vào giữa; có thông báo + selector pet thì CTA chưa hiện ngay khi mở màn. Không ghi nhận kiểm tra này là đạt yêu cầu luôn nằm sẵn trong viewport.
- Test lưu lại: `python -I scripts/smoke-pet-layout.py`, cần Python Playwright và stack local `vpet-ui-test` cổng 18099; chỉ tạo tài khoản test, giả lập response trạng thái biên, không sửa DB production. Dọn stack bằng compose `down -v` sau chạy.
- Frontend 78/78, Go suite và Docker build đạt; DB repository integration vẫn tự skip, webpack còn ba cảnh báo kích thước (~738 KiB JS). Skill max/thiếu vật liệu, toàn bộ trạng thái giá sai/thiếu vàng và buy-order UI chưa E2E đầy đủ trong đợt này.
- Đã push/deploy app `93c93fc` ngày 31/10/2026. App/DB healthy; backup DB đã qua `pg_restore -l`, giữ image rollback riêng. Origin/public/local cùng bundle 755.756 byte, SHA-256 `9d749eaae9ac5464f2b8277aadd3693fbb54797a196f9392f2d774a37101c0b2`; 43 ảnh runtime public hợp lệ. Login public 1440/375 không pageerror/tràn ngang. Stack/volume và fixture test local đã dọn; task gameplay cũ giữ nguyên.

## Mobile / âm thanh / Halloween

- Đã viết: header mobile một hàng, tiện ích gom menu; Thú cưng chia Chăm sóc/Trang bị/Phát triển, Chợ chia Xem chợ/Đăng tin/Tin của tôi. Landscape đổi cấu trúc controls, CTA ưu tiên hiển thị trước; giữ cảnh 16:9 và nút tối thiểu 44px.
- Đã viết: ba arrangement Web Audio tự soạn cho thành phố/phiêu lưu/trận đấu, SFX theo lượt chiến đấu/gacha/thao tác thành công; nhạc và tiếng động có opt-in/âm lượng riêng. Scheduler giới hạn voice, dừng khi ẩn tab/unmount; không thêm thư viện hoặc nhạc bên ngoài.
- Đã viết: Halloween mở từ 31/10 đến hết 01/11/2026 theo giờ Việt Nam (đóng lúc 00:00 ngày 02/11); đổi vĩnh viễn một thú sống để hồi sinh thú đã chôn. Lưu snapshot khi chôn mới, giữ cấp/kỹ năng/trang bị/cường hóa/ngọc/phụ kiện. Đồ vốn trả về kho khi chôn nên phải thu lại đúng món; thiếu đồ từ chối toàn bộ, không nhân bản. Chặn thú nghỉ dưỡng, lời mời tỷ thí và tổ đội chờ.
- Giới hạn dữ liệu: mộ trước bản này chỉ có tên/ngoại hình, không có lịch sử cấp/kỹ năng/đồ; chỉ hồi sinh cấp 1, không bịa dữ liệu. Thú hiến tế không tạo mộ, đồ còn lại trả kho. Không thưởng hoạt động khi hồi sinh.
- Popup sự kiện dùng cờ từ server, hiện một lần mỗi tài khoản/năm trên trình duyệt; ưu tiên quà/điểm danh trước, Escape và focus bằng native dialog; CTA đến Nghĩa trang. Có vi/en/ja, cảnh báo hiến tế không hoàn tác và giới hạn mộ cũ.
- Đã kiểm thử: Go toàn suite; test hồi sinh bảo toàn đồ, từ chối không mutate, lỗi commit không đổi DB, legacy và ranh giới múi giờ. HTTP local thật hồi sinh đủ đồ +3/ngọc/phụ kiện, từ chối gửi lại và giữ kết quả sau reload. Sáu repository integration test đã chạy thật trên DB local cô lập, không skip.
- Đã kiểm thử: frontend 81/81 và build đạt; fake AudioContext kiểm tra opt-in, một context, cue, mute/ẩn tab/cleanup/resume sau dispose. Playwright bắt pageerror: popup một lần/Escape/CTA, xác nhận hiến tế, nhóm pet/chợ, 360/375/390/768/1440 và landscape 740×375/844×390 đạt; đã xem screenshot và sửa lỗi co ngang pet. CTA landscape hiển thị ngay, không cần scrollIntoView.
- Chưa xác nhận chất lượng nghe bằng tai; nhạc vẫn là tổng hợp Web Audio, không phải bản thu orchestral. Chưa E2E đầy đủ pause/skip/replay/gacha/audio lifecycle trên mọi browser; nhãn ải landscape vẫn dày. Task gameplay cũ phía trên giữ nguyên.
- Đã push/deploy app `380282d` ngày 31/10/2026; app/DB healthy. Backup DB đã qua `pg_restore -l`, giữ image rollback riêng. Bundle local đã test/origin/public cùng 779.715 byte, SHA-256 `022108fea5ad919dca357cdc296413b60869403447a9e6ee4d6899f4c412a48b`; public xác minh bằng browser vì HTTP client Python bị Cloudflare trả 403. Màn đăng nhập public 1440/375 không pageerror/tràn ngang; không tạo tài khoản production.
- Smoke bổ sung đạt: bật/tắt nhạc và SFX bằng gesture, lưu âm lượng, menu Escape; CTA landscape có toast + hai pet vẫn trên nav. Regression 6 trạng thái pet (hai/một/chết/nghỉ dưỡng/kho trống/no-pet) ở 1440/1024/375 đạt. Stack/volume test local đã dọn, không đụng volume production.

## Sửa cuộn mobile / gia hạn Halloween 31/10/2026

- Đã viết: bỏ `body overflow:hidden` còn sót từ modal cũ; cuộn dọc native trở lại, giữ cuộn nội dung riêng ở landscape. Gia hạn sự kiện đến hết 01/11/2026 theo giờ Việt Nam; server, popup và Nghĩa trang đồng bộ vi/en/ja, không tự mở lại năm sau.
- Đã kiểm thử: frontend 81/81, build đạt (ba cảnh báo kích thước); Go toàn suite đạt, store integration đợt sửa này không bật. Hồi sinh có snapshot và legacy thực thi ngày 01/11; kiểm tra trước/sau hai mốc mở/đóng và năm 2027.
- Playwright có touch emulation + CDP touchStart/touchMove/touchEnd: cuộn thật ở 360/375/390/768 và landscape 740/844; khôi phục CSS khóa cũ thì không cuộn, gỡ khóa thì cuộn được. Vuốt tới ô chat Thành phố ở ba kích thước phone; CTA landscape không bị nav che, không pageerror/tràn ngang. Popup hiển thị đúng hạn mới. Chưa thử trực tiếp thiết bị iOS/Android vật lý.
- Deploy: chờ phát hành bản sửa; không coi bản `380282d` đã chứa thay đổi này.

## Kiểm chứng artwork production

- Bản app: `4fac173`, 31/10/2026; không thay backend hoặc dữ liệu người chơi.
- Origin/public `main.js` cùng SHA-256 `ac9124ee7b4e564c658b71790085196d5afca987c67b56700e9a9a4d392e83e6`, 746.033 byte.
- 37 tài nguyên PNG/WebP được bundle tham chiếu tải public thành công, đúng chữ ký ảnh; `Cache-Control: no-cache`, Cloudflare `REVALIDATED`.
- Backup DB định dạng custom đã qua `pg_restore -l`; image rollback giữ riêng. Stack/volume test local đã dọn.
- Frontend 78/78; 11 kiểm tra màn/kích thước local đạt. Store integration và E2E gameplay chi tiết chưa chạy trong đợt artwork.

## Tham chiếu UI bổ sung #5–#7

- Ảnh #5: cảm giác chăm thú trong thế giới sống; tham khảo pet selector và hành động, không lặp cả bản đồ Thành phố trong màn Thú cưng.
- Ảnh #7: bố cục Thú cưng chính — một bộ chọn pet, cảnh pet + trang bị, chăm sóc/đồ dùng, kỹ năng/tiềm năng/chỉ số, bộ đồ/XP. Reuse `PetTab`, `GearBox`, `ItemGrid`, `Skills`, `Build`, `StatSheet`; không đổi API chăm sóc.
- Ảnh #6: bố cục Nhiệm vụ chính — nhóm trái, nhiệm vụ giữa, gợi ý phải; mobile tab và dòng hai hàng. Reuse `Row`, `Titles`, `ladders`.
- Nút nhiệm vụ: Thực hiện chỉ điều hướng đúng đích; Nhận thưởng gọi claim khi đủ; Đã nhận disabled. Mapping theo ID thật, không theo tên dịch.
- Không thêm rương mốc 2/4/6, thưởng hoàn thành tất cả, cấp nhà, giới tính hoặc tăng cấp thủ công chỉ vì xuất hiện trong ảnh. Mọi số liệu/giới hạn/chi phí lấy từ game thật.
- Nav xanh/panel sáng trong ảnh là tham khảo phân cấp; chưa thay hướng nav gỗ chung đã duyệt. Giữ selected vàng và một bộ icon/trạng thái xuyên các màn; không tạo hai theme.
- Asset bổ sung: cảnh sân chăm pet, bệ pet, icon care/quest/skill và slot gear đồng bộ. Giữ pet thật bằng `PetCanvas`.
- Nghiệm thu thêm: đổi pet tác động đúng đối tượng, pet bệnh/chết/no-pet, kho trống, skill max/thiếu vật liệu; nhiệm vụ đủ/chưa đủ/đã nhận, Thành tựu/Danh hiệu, vi/en/ja, desktop/mobile. Đã smoke bố cục Thành phố/Nhiệm vụ/Thú cưng ở ba kích thước; các trạng thái gameplay chi tiết trong dòng này chưa E2E đầy đủ.

## Thứ tự tiếp tục

1. Rà thay đổi đang dở; hoàn thiện thông báo thưởng tháp và test gear còn thiếu.
2. Chốt asset hợp lệ. Làm vỏ + Thành phố + Nhiệm vụ; screenshot, sửa trước khi áp các màn khác.
3. Kiểm tra chức năng và mỹ thuật riêng: desktop 1672×941/1440×900/1024×768, mobile 375×740, landscape 740×375/844×390.
4. Chạy Go, frontend, DB integration cô lập, Playwright bắt `pageerror`; test no-pet, i18n, reload, SSE, giao dịch.
5. Dọn test local đúng phạm vi; rà diff và file public; commit/push/deploy bản hoàn chỉnh theo yêu cầu.

## Điểm cần chú ý

- Giữ boss chiến dịch ải 6 và 12; vùng cũ vẫn mở sau ải 6, không khóa tiến độ cũ sau mở rộng.
- Tower dùng phần vàng còn lại của trần hoạt động, không nhầm `payout` là hàm giới hạn ngày.
- `/use` sách phải chọn nguồn/đích; không phục hồi random fallback cho client cũ.
- Tinh Hồn chiến tranh là nguồn tiến trình vĩnh viễn mới, khác thiết kế buff mùa ban đầu; test giới hạn và nguyên tử rõ ràng.
- Các test đang xanh trước thay đổi cuối không thay thế xác minh bản phát hành.
- Nếu thiếu tranh phù hợp: để task asset pending, không gọi nền low-poly/gradient là UI hoàn tất.
