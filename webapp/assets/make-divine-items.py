from pathlib import Path
import sys

from PIL import Image

SOURCE_FILES = [
    ('kiemthanlong', 'sword.png'),
    ('giapthanlong', 'platemail.png'),
    ('muthiende', 'platemail_helmet.png'),
    ('gangthienloi', 'gauntlets.png'),
    ('hiathienvuong', 'greaves.png'),
    ('nhanthienmenh', 'gold_ring.png'),
    ('khienthienha', 'tower_shield.png'),
    ('aothienmenh', 'cloak.png'),
]
SIZE = 32


def recolor(image, lightning=False):
    image = image.convert('RGBA')
    out = Image.new('RGBA', image.size)
    pixels = []
    for red, green, blue, alpha in image.getdata():
        if alpha == 0:
            pixels.append((0, 0, 0, 0))
            continue
        value = max(red, green, blue) / 255
        if lightning and blue > red * 1.2 and blue > green * 1.05:
            pixels.append((80, 205, 255, alpha))
        elif value > .72:
            pixels.append((255, int(166 + value * 70), int(45 + value * 40), alpha))
        elif value > .35:
            pixels.append((int(112 + value * 90), int(16 + value * 35), int(20 + value * 25), alpha))
        else:
            pixels.append((int(35 + value * 75), int(4 + value * 18), int(9 + value * 22), alpha))
    out.putdata(pixels)
    return out


def main():
    if len(sys.argv) != 3:
        raise SystemExit('usage: make-divine-items.py SOURCE_DIR OUTPUT_DIR')
    source = Path(sys.argv[1]).resolve()
    output = Path(sys.argv[2]).resolve()
    output.mkdir(parents=True, exist_ok=True)
    for item_id, filename in SOURCE_FILES:
        with Image.open(source / filename) as opened:
            image = opened.convert('RGBA')
        image.thumbnail((28, 28), Image.Resampling.NEAREST)
        canvas = Image.new('RGBA', (SIZE, SIZE))
        canvas.alpha_composite(image, ((SIZE - image.width) // 2, (SIZE - image.height) // 2))
        recolor(canvas, item_id == 'gangthienloi').save(output / f'{item_id}.png')


if __name__ == '__main__':
    main()
