# VPet standalone

Game web độc lập tách từ plugin `shourenso.vpet`. Go phục vụ API + React bundle; PostgreSQL giữ tài khoản, session, game state, multiplayer và notification.

## Chạy local

```bash
cp .env.example .env
# thay POSTGRES_PASSWORD và TRANSFER_SECRET
# local HTTP giữ COOKIE_SECURE=false
docker compose --env-file .env up --build
```

Mở `http://localhost:8080`. Production phải đặt reverse proxy TLS, `PUBLIC_URL=https://...`, `COOKIE_SECURE=true`.

## Kiểm tra

```bash
docker run --rm -v "$PWD:/src" -w /src golang:1.25.9 go test ./...
cd webapp && npm ci && npm test && npm run build
```

Store integration tests tự skip nếu chưa có DB. Chạy với PostgreSQL thử riêng:

```bash
VPET_TEST_DATABASE_URL='postgres://vpet:password@127.0.0.1:5432/vpet?sslmode=disable' \
  go test ./internal/store -count=1
```

Mỗi test tạo schema riêng rồi xóa; không trỏ biến này vào production.

## Snapshot migration

Snapshot JSON schema 1:

```json
{
  "schema": 1,
  "batch": "cutover-2026-10-30",
  "exported_at": "2026-10-30T00:00:00Z",
  "users": [{"id":"mattermost-user-id","username":"user","display_name":"User","locale":"vi","disabled":false}],
  "kv": {"u_mattermost-user-id": {}},
  "checksums": {"u_mattermost-user-id": "sha256-of-raw-json-value"}
}
```

Validate offline:

```bash
docker compose run --rm --entrypoint vpet-migrate app -file /snapshot.json validate
```

Import đầu tiên chỉ nhận database trống. Chạy lại cùng batch/snapshot sẽ xác minh rồi thành công; batch trùng nhưng dữ liệu khác bị từ chối:

```bash
docker compose run --rm --entrypoint vpet-migrate app \
  -database "$DATABASE_URL" -file /snapshot.json import
```

Mã chuyển:

```text
v1.<base64url(batch)>.<base64url(mattermost-user-id)>.<hex-hmac-sha256>
```

MAC dùng cùng `TRANSFER_SECRET`. Mã gắn đúng user ID và batch; claim transaction khóa row, dùng một lần.

## Cutover

1. Dry-run snapshot, validate, import vào DB thử; so counts/checksums.
2. Backup Mattermost DB/KV và standalone DB.
3. Bật migration mode trong plugin để chặn ghi.
4. Xuất snapshot cuối, validate, import vào DB sạch.
5. Smoke account claim + gameplay + market/duel/boss/party.
6. Mở standalone. Chỉ sau kiểm chứng mới disable plugin.

Không xóa Mattermost plugin KV, volume hoặc `data/`. Nếu standalone đã nhận giao dịch mới, không bật lại plugin cũ mà chưa reconcile vì hai nguồn dữ liệu đã phân kỳ.

## Backup

```bash
docker compose --env-file .env exec -T db pg_dump -U vpet -d vpet -Fc > vpet.dump
docker compose --env-file .env exec -T db pg_restore --clean --if-exists -U vpet -d vpet < vpet.dump
```

## Thưởng hoạt động

Hành động thành công thuộc ba nhóm `care`, `adventure`, `daily`. Mỗi nhóm có hồi chiêu riêng; tổng thưởng vẫn khóa ở 70 vàng/ngày và tối đa 6 túi tiếp tế/ngày. Request lỗi, mua/bán và claim thưởng không cấp thêm vàng.

## Giới hạn bản đầu

- Password reset bằng lệnh quản trị; chưa có email/OAuth.
- Notification dùng SSE polling DB 2 giây.
- App node hiện giữ mutex gameplay. PostgreSQL transaction bảo vệ ghi nhiều key, nhưng read-modify-write vẫn giả định một app node. Chỉ scale ngang sau khi chuyển toàn bộ mutation vào transaction repository.
