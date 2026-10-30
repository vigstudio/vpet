#!/usr/bin/env python3
"""Build the small, redistributable VPet chrome from the supplied CraftPix ZIP."""

import argparse
import os
import struct
import zlib
from pathlib import Path
from zipfile import ZipFile

PNG = b'\x89PNG\r\n\x1a\n'
# Output name -> source sheet and crop. These are UI pieces, never complete source sheets.
CROPS = {
    'rpg-panel.png': ('PNG/Inventory.png', (7, 0, 98, 101)),
    'rpg-button.png': ('PNG/Buttons.png', (16, 451, 128, 12)),
    'rpg-slot.png': ('PNG/Inventory.png', (125, 16, 18, 18)),
    'rpg-inventory.png': ('PNG/Inventory.png', (7, 0, 98, 101)),
    'rpg-equipment.png': ('PNG/Equipment.png', (7, 0, 146, 101)),
    'rpg-shop.png': ('PNG/Shop.png', (7, 0, 98, 149)),
    'rpg-craft.png': ('PNG/Craft.png', (8, 0, 200, 160)),
    'rpg-settings.png': ('PNG/Settings.png', (7, 0, 98, 149)),
    'rpg-result.png': ('PNG/Win_loose.png', (8, 8, 96, 88)),
    'rpg-icons.png': ('PNG/Icons.png', (0, 0, 96, 304)),
}


def paeth(a, b, c):
    p = a + b - c
    pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
    return a if pa <= pb and pa <= pc else b if pb <= pc else c


def decode_png(data):
    if not data.startswith(PNG):
        raise ValueError('not a PNG')
    pos, stream, header = 8, bytearray(), None
    while pos < len(data):
        size = struct.unpack('>I', data[pos:pos + 4])[0]
        kind = data[pos + 4:pos + 8]
        body = data[pos + 8:pos + 8 + size]
        pos += size + 12
        if kind == b'IHDR':
            header = struct.unpack('>IIBBBBB', body)
        elif kind == b'IDAT':
            stream.extend(body)
    width, height, depth, color, compression, filtering, interlace = header
    if (depth, color, compression, filtering, interlace) != (8, 6, 0, 0, 0):
        raise ValueError(f'expected non-interlaced 8-bit RGBA PNG, got {header}')
    raw, stride, rows, at = zlib.decompress(stream), width * 4, [], 0
    prior = bytearray(stride)
    for _ in range(height):
        mode = raw[at]
        row = bytearray(raw[at + 1:at + 1 + stride])
        at += stride + 1
        for i in range(stride):
            left = row[i - 4] if i >= 4 else 0
            up = prior[i]
            upper_left = prior[i - 4] if i >= 4 else 0
            predictors = (0, left, up, (left + up) // 2, paeth(left, up, upper_left))
            if mode > 4:
                raise ValueError(f'unsupported PNG filter {mode}')
            row[i] = (row[i] + predictors[mode]) & 255
        rows.append(row)
        prior = row
    return width, height, rows


def chunk(kind, body):
    return struct.pack('>I', len(body)) + kind + body + struct.pack('>I', zlib.crc32(kind + body) & 0xffffffff)


def encode_png(width, rows):
    raw = b''.join(b'\0' + bytes(row) for row in rows)
    header = struct.pack('>IIBBBBB', width, len(rows), 8, 6, 0, 0, 0)
    return PNG + chunk(b'IHDR', header) + chunk(b'IDAT', zlib.compress(raw, 9)) + chunk(b'IEND', b'')


def crop(data, box):
    width, height, rows = decode_png(data)
    x, y, out_width, out_height = box
    if x < 0 or y < 0 or x + out_width > width or y + out_height > height:
        raise ValueError(f'crop {box} outside {width}x{height}')
    out = [row[x * 4:(x + out_width) * 4] for row in rows[y:y + out_height]]
    if not any(pixel for row in out for pixel in row[3::4]):
        raise ValueError(f'crop {box} is transparent')
    return encode_png(out_width, out)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('archive', nargs='?', default=os.environ.get('CRAFTPIX_RPG_UI_ZIP'))
    parser.add_argument('--output', default=Path(__file__).resolve().parent)
    args = parser.parse_args()
    if not args.archive:
        parser.error('pass the CraftPix ZIP or set CRAFTPIX_RPG_UI_ZIP')
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    with ZipFile(args.archive) as archive:
        names = set(archive.namelist())
        for out_name, (source, box) in CROPS.items():
            if source not in names:
                raise ValueError(f'missing {source}')
            (output / out_name).write_bytes(crop(archive.read(source), box))
    print('generated ' + ', '.join(CROPS))


if __name__ == '__main__':
    main()
