#!/usr/bin/env python3
"""DM each player their one-time transfer code from the Thú ảo bot.

    MM_TOKEN=... ./send-claim-dm.py codes.tsv [--limit N] [--only user-id] [--dry-run]

Each code is a bearer credential, so one DM per recipient and never a group
post. Already-messaged user ids are appended to <codes>.sent and skipped on a
rerun, which makes an interrupted send safe to repeat.
"""
import json
import os
import sys
import time
import urllib.error
import urllib.request

BASE = os.environ.get("MM_URL", "http://127.0.0.1:11113")
SITE = os.environ.get("VPET_URL", "https://vpet.nghiane.com")
TOKEN = os.environ.get("MM_TOKEN", "").strip()


def api(path, payload=None, method=None):
    data = json.dumps(payload).encode() if payload is not None else None
    request = urllib.request.Request(
        f"{BASE}/api/v4{path}",
        data=data,
        method=method or ("POST" if data else "GET"),
        headers={"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def message(username, code):
    return f"""### 🐾 Thú ảo chuyển sang trang riêng

Chào **{username}**, Thú ảo giờ có trang riêng tại {SITE} — không cần mở SHourenso nữa, chơi được cả trên điện thoại.

Toàn bộ thú, vàng, đồ, danh hiệu và tiến độ của bạn đã được chuyển sẵn. Lấy lại bằng mã dưới đây.

**Hướng dẫn "Chuyển dữ liệu":**

1. Mở {SITE}
2. Chọn thẻ **Chuyển dữ liệu** (thẻ thứ ba, cạnh Đăng nhập và Tạo mới)
3. Dán **mã chuyển một lần** của bạn:

```
{code}
```

4. Tên đăng nhập: điền đúng `{username}`
5. Đặt **mật khẩu mới** cho trang Thú ảo (tối thiểu 10 ký tự — đây là mật khẩu riêng, không phải mật khẩu SHourenso)
6. Bấm **Nhận dữ liệu và vào game**

Xong. Những lần sau chỉ cần đăng nhập bằng tên và mật khẩu vừa đặt.

⚠️ Mã chỉ dùng được **một lần** và chỉ dành riêng cho bạn — đừng chia sẻ cho ai.
Plugin Thú ảo trong SHourenso đã ngừng; tiến độ mới chỉ được ghi nhận trên trang riêng.

---

### 🐾 Virtual Pet has moved to its own site

Hi **{username}**, Virtual Pet now lives at {SITE}. All your pets, gold, gear and progress have been transferred.

**How to transfer:** open {SITE} → **Chuyển dữ liệu** tab → paste the one-time code above → username `{username}` → choose a new password (10+ characters) → **Nhận dữ liệu và vào game**.

The code works only once and is yours alone. Please do not share it."""


def main():
    if not TOKEN:
        sys.exit("MM_TOKEN is required")
    path = sys.argv[1]
    dry_run = "--dry-run" in sys.argv
    only = None
    limit = None
    if "--only" in sys.argv:
        only = sys.argv[sys.argv.index("--only") + 1]
    if "--limit" in sys.argv:
        limit = int(sys.argv[sys.argv.index("--limit") + 1])

    me = api("/users/me")
    bot_id = me["id"]
    sent_path = path + ".sent"
    done = set()
    if os.path.exists(sent_path):
        with open(sent_path, encoding="utf-8") as handle:
            done = {line.strip() for line in handle if line.strip()}

    with open(path, encoding="utf-8") as handle:
        rows = [line.rstrip("\n").split("\t") for line in handle if line.strip()]

    ok = failed = skipped = 0
    with open(sent_path, "a", encoding="utf-8") as log:
        for user_id, username, code in rows:
            if only and user_id != only:
                continue
            if user_id in done:
                skipped += 1
                continue
            if limit is not None and ok >= limit:
                break
            if dry_run:
                print(f"DRY {username}")
                ok += 1
                continue
            try:
                channel = api("/channels/direct", [bot_id, user_id])
                api("/posts", {"channel_id": channel["id"], "message": message(username, code)})
                log.write(user_id + "\n")
                log.flush()
                ok += 1
            except urllib.error.HTTPError as error:
                failed += 1
                print(f"FAIL {username} {error.code} {error.read()[:120]!r}", file=sys.stderr)
            except Exception as error:  # noqa: BLE001 - report and continue the batch
                failed += 1
                print(f"FAIL {username} {error}", file=sys.stderr)
            time.sleep(0.15)

    print(f"sent={ok} failed={failed} skipped={skipped}")


if __name__ == "__main__":
    main()
