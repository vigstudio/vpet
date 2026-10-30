#!/usr/bin/env python3
"""Derive the one-time transfer code for every user in a snapshot.

    TRANSFER_SECRET=<hex> ./claim-codes.py snapshot.json > codes.tsv

Writes "user-id<TAB>username<TAB>code" and keeps the secret out of argv.
Codes are bearer credentials: never log or commit the output.
"""
import base64
import hashlib
import hmac
import json
import os
import sys


def b64(text):
    return base64.urlsafe_b64encode(text.encode()).decode().rstrip("=")


def main():
    secret = os.environ.get("TRANSFER_SECRET", "").strip()
    try:
        # The server hex-decodes TRANSFER_SECRET, so the MAC key is the raw bytes.
        key = bytes.fromhex(secret)
    except ValueError:
        sys.exit("TRANSFER_SECRET must be hex")
    if len(key) < 32:
        sys.exit("TRANSFER_SECRET must decode to at least 32 bytes")
    with open(sys.argv[1], encoding="utf-8") as handle:
        snapshot = json.load(handle)
    for user in snapshot["users"]:
        payload = f"v1.{b64(snapshot['batch'])}.{b64(user['id'])}"
        mac = hmac.new(key, payload.encode(), hashlib.sha256).hexdigest()
        print(f"{user['id']}\t{user['username']}\t{payload}.{mac}")


if __name__ == "__main__":
    main()
