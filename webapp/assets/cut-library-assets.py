"""Cut image #21; keep source coordinates and do not ship sample text as UI.

Usage: python cut-library-assets.py path/to/21.png
Requires Pillow. Output is separate from the larger fantasy kit.
"""
import argparse
import hashlib
import json
import shutil
from collections import deque
from pathlib import Path

from PIL import Image, ImageDraw


def cut_background(image):
    image = image.convert('RGBA')
    pixels = image.load()
    width, height = image.size
    queue, seen = deque(), set()

    def visit(x, y):
        if (x, y) in seen:
            return
        seen.add((x, y))
        r, g, b, a = pixels[x, y]
        # ponytail: this sheet has a dark purple backdrop; other sheets need
        # their own mask rather than treating every dark outline as background.
        if a == 0 or (r < 72 and g < 61 and b < 119):
            pixels[x, y] = (r, g, b, 0)
            queue.append((x, y))

    for x in range(width):
        visit(x, 0)
        visit(x, height - 1)
    for y in range(height):
        visit(0, y)
        visit(width - 1, y)
    while queue:
        x, y = queue.popleft()
        for xx, yy in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
            if 0 <= xx < width and 0 <= yy < height:
                visit(xx, yy)
    box = image.getbbox()
    if not box:
        raise ValueError('Empty crop')
    return image.crop(box)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    args = parser.parse_args()
    base = Path(__file__).parent
    source = Image.open(args.source).convert('RGBA')
    if source.size != (1536, 1024):
        raise ValueError('Expected the original 1536x1024 image #21')
    output = base / 'library'
    output.mkdir(exist_ok=True)
    saved_source = base / 'fantasy-source' / '21.png'
    if args.source.resolve() != saved_source.resolve():
        shutil.copyfile(args.source, saved_source)
    specs = []

    def row(names, edges, top, bottom, remove=True):
        assert len(names.split()) == len(edges) - 1
        for name, left, right in zip(names.split(), edges, edges[1:]):
            specs.append((name, (left, top, right, bottom), remove))

    row('nav-city nav-pet nav-quest nav-world nav-hunt', [28, 95, 169, 242, 313, 389], 145, 200)
    row('nav-bag nav-shop nav-clinic nav-market nav-duel nav-grave', [32, 94, 168, 241, 305, 360, 398], 231, 280)
    row('coin gem energy heart happy', [29, 91, 162, 232, 300, 367], 377, 429)
    row('clover star crown trophy medal gift', [29, 95, 156, 217, 276, 332, 389], 434, 485)
    row('pet-orange pet-purple pet-blue pet-yellow pet-pink pet-green', [443, 529, 633, 731, 824, 922, 1007], 144, 239)
    row('building-home building-hunt building-portal building-arena building-clinic', [436, 554, 659, 779, 913, 1022], 377, 494)
    row('building-shop building-warehouse building-market building-grave building-fountain', [435, 553, 673, 786, 915, 1023], 521, 613)
    row('food-steak food-sushi food-cake food-tea food-apple food-grape item-gold item-stone item-wood item-crystal', [447, 508, 569, 624, 680, 737, 796, 848, 905, 960, 1014], 683, 729)
    row('weapon-sword gear-shield gear-armor gear-boots gear-ring gear-cloak potion-red potion-blue item-key item-scroll', [447, 508, 569, 624, 680, 737, 796, 848, 905, 960, 1014], 731, 778)
    row('chest-wood chest-silver chest-gold chest-purple chest-open', [436, 498, 556, 616, 679, 768], 829, 890)
    row('effect-star effect-sparkle effect-heart effect-lightning effect-level', [790, 844, 891, 936, 983, 1024], 830, 888)
    row('badge-medal badge-crown badge-shield badge-star badge-fire badge-ice badge-energy badge-leaf', [435, 477, 522, 564, 609, 653, 697, 740, 782], 935, 980)
    for name, box in [
        ('scene-sky', (1075, 151, 1270, 188)),
        ('scene-meadow', (1299, 146, 1510, 193)),
        ('scene-island', (1067, 209, 1280, 255)),
        ('scene-night', (1299, 209, 1510, 255)),
        ('scene-sunset', (1067, 272, 1280, 318)),
        ('scene-cave', (1299, 272, 1510, 318)),
        ('scene-snow', (1067, 335, 1280, 378)),
        ('scene-desert', (1299, 335, 1510, 378)),
    ]:
        specs.append((name, box, False))
    specs.extend([
        ('island-large', (1065, 441, 1265, 544), True),
        ('island-small', (1271, 443, 1357, 502), True),
        ('pine-large', (1366, 423, 1417, 498), True),
        ('tree', (1417, 436, 1475, 498), True),
        ('pine-small', (1476, 437, 1518, 498), True),
        ('rocks', (1257, 505, 1314, 545), True),
        ('fence', (1323, 505, 1402, 545), True),
        ('bush-pink', (1404, 505, 1453, 545), True),
        ('bush-yellow', (1459, 505, 1515, 545), True),
        ('panel-ribbon', (17, 864, 148, 999), True),
        ('panel-paper', (155, 859, 275, 932), True),
        ('panel-wood', (279, 857, 398, 925), True),
        ('panel-stone', (155, 935, 275, 999), True),
        ('panel-purple', (279, 931, 398, 999), True),
    ])
    manifest = {'source': 'fantasy-source/21.png', 'sha256': hashlib.sha256(args.source.read_bytes()).hexdigest(), 'assets': {}}
    preview = Image.new('RGB', (960, ((len(specs) + 7) // 8) * 130), '#fff0d2')
    draw = ImageDraw.Draw(preview)
    for i, (name, box, remove) in enumerate(specs):
        image = source.crop(box)
        if remove:
            image = cut_background(image)
            assert image.getextrema()[3] == (0, 255), name
        image.save(output / (name + '.png'), optimize=True)
        with Image.open(output / (name + '.png')) as saved:
            assert saved.mode == 'RGBA' and saved.size == image.size
        manifest['assets'][name] = {'crop': box, 'size': image.size, 'background_removed': remove}
        thumb = image.copy()
        thumb.thumbnail((110, 96), Image.Resampling.LANCZOS)
        x, y = i % 8 * 120, i // 8 * 130
        preview.paste(thumb, (x + (120 - thumb.width) // 2, y), thumb)
        draw.text((x + 3, y + 100), name, fill='#213653')
    (output / 'manifest.json').write_bytes((json.dumps(manifest, indent=2) + '\n').encode('utf-8'))
    preview.save(output / 'preview.jpg', quality=92)
    print(f'{len(specs)} assets: {output}')


if __name__ == '__main__':
    main()
