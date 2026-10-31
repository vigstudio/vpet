"""Cut the user-provided UI sheets. Usage: python cut-fantasy-assets.py ui.png icons.png

Requires Pillow, like the other asset-generation scripts in this directory.
Only boundary-connected near-black pixels are removed, preserving interior outlines.
"""
import argparse
import hashlib
import json
import shutil
from collections import deque
from pathlib import Path

from PIL import Image, ImageDraw


def transparent_background(image):
    image = image.convert('RGBA')
    if image.getextrema()[3][0] == 0:
        return image.crop(image.getbbox())
    pixels = image.load()
    width, height = image.size
    queue = deque()
    seen = set()

    def visit(x, y):
        if (x, y) in seen:
            return
        seen.add((x, y))
        r, g, b, _ = pixels[x, y]
        if max(r, g, b) <= 38:
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
        raise ValueError('Crop contains no artwork')
    return image.crop(box)


def blank_button(image):
    # ponytail: these six buttons have uniform horizontal centers; use a clean
    # column to remove the baked paw. Nonuniform future buttons need clean art.
    width, height = image.size
    image.paste(image.crop((39, 0, 40, height)).resize((width - 80, height)), (40, 0))
    return image


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('ui', type=Path)
    parser.add_argument('icons', type=Path)
    parser.add_argument('--extras-dir', type=Path, help='Directory containing sheets 14.png through 18.png')
    parser.add_argument('--out', type=Path, default=Path(__file__).parent / 'fantasy')
    parser.add_argument('--source-dir', type=Path, default=Path(__file__).parent / 'fantasy-source')
    args = parser.parse_args()
    ui = Image.open(args.ui).convert('RGBA')
    icons = Image.open(args.icons).convert('RGBA')
    if ui.size != (1448, 1086) or icons.size != (1448, 1086):
        raise ValueError('Expected the original 1448x1086 sheets')
    args.out.mkdir(parents=True, exist_ok=True)
    specs = []

    def add(sheet, name, box, blank=False):
        specs.append((sheet, name, box, blank))

    nav = [
        ('city', 6, 173), ('pet', 174, 316), ('quest', 317, 438),
        ('world', 439, 565), ('hunt', 566, 684), ('bag', 685, 801),
        ('shop', 802, 942), ('clinic', 943, 1083), ('market', 1084, 1197),
        ('duel', 1198, 1305), ('grave', 1306, 1444),
    ]
    for name, left, right in nav:
        add(icons, 'nav-' + name, (left, 30, right, 228))
    for name, box in [
        ('coin', (27, 300, 209, 473)), ('gem', (214, 300, 387, 473)),
        ('heart', (392, 300, 566, 473)), ('energy', (573, 300, 711, 473)),
        ('shield', (722, 300, 887, 473)), ('paw', (899, 300, 1072, 473)),
        ('trophy', (1077, 300, 1250, 473)), ('crown', (1258, 300, 1442, 473)),
        ('map', (20, 483, 185, 635)), ('mail', (185, 483, 325, 635)),
        ('settings', (326, 483, 461, 635)), ('music', (462, 483, 589, 635)),
        ('bell', (590, 483, 714, 635)), ('search', (715, 483, 860, 635)),
        ('close', (861, 483, 993, 635)), ('check', (994, 483, 1130, 635)),
        ('plus', (1131, 483, 1267, 635)), ('chat', (1268, 483, 1442, 635)),
        ('warning', (54, 639, 269, 834)), ('timer', (280, 639, 469, 834)),
        ('scroll', (479, 639, 677, 834)), ('egg', (691, 639, 885, 834)),
        ('portal', (897, 639, 1144, 834)), ('backpack', (1152, 639, 1392, 834)),
    ]:
        add(icons, name, box)
    for name, left, right in [
        ('cat', 20, 270), ('fox', 275, 500), ('rabbit', 510, 724),
        ('dragon', 726, 958), ('penguin', 960, 1200), ('cat-flower', 1201, 1444),
    ]:
        add(icons, 'mascot-' + name, (left, 834, right, 1083))
    for name, box in [
        ('gold', (50, 20, 282, 111)), ('blue', (283, 20, 490, 111)),
        ('green', (491, 20, 698, 111)), ('disabled', (699, 20, 893, 111)),
        ('red', (894, 20, 1097, 111)), ('paper', (1098, 20, 1280, 111)),
    ]:
        add(ui, 'button-' + name, box, True)
    for name, box in [
        ('frame-wood', (19, 213, 354, 424)),
        ('frame-blue', (362, 220, 719, 423)),
        ('frame-gold', (725, 209, 1092, 424)),
        ('frame-leaf', (1099, 209, 1428, 424)),
        ('divider-leaf', (485, 965, 745, 1020)),
        ('divider-gold', (494, 1024, 732, 1073)),
        ('corner-leaf', (1006, 947, 1110, 1065)),
        ('corner-gold', (1113, 947, 1209, 1065)),
        ('corner-blue', (1211, 947, 1306, 1065)),
        ('corner-wood', (1307, 947, 1427, 1065)),
    ]:
        add(ui, name, box)

    if args.extras_dir:
        sheets = {n: Image.open(args.extras_dir / f'{n}.png').convert('RGBA') for n in range(14, 19)}
        if any(sheet.size != (1448, 1086) for sheet in sheets.values()):
            raise ValueError('Expected original 1448x1086 extra sheets')

        def row(sheet, names, edges, top, bottom):
            for name, left, right in zip(names.split(), edges, edges[1:]):
                add(sheets[sheet], name, (left, top, right, bottom))

        row(14, 'skill-fire skill-shield skill-heal skill-lightning skill-ice skill-poison',
            [0, 285, 506, 719, 950, 1178, 1448], 20, 320)
        row(14, 'skill-wind skill-water skill-earth skill-light skill-curse skill-speed',
            [0, 272, 503, 748, 951, 1190, 1448], 321, 610)
        row(14, 'status-burn status-freeze status-stun status-sleep status-regen status-attack status-defense status-critical status-dodge status-target',
            [0, 158, 305, 476, 629, 786, 922, 1043, 1199, 1312, 1448], 618, 818)
        row(14, 'effect-sparkle effect-hit effect-magic effect-dust effect-level effect-love',
            [0, 210, 438, 670, 908, 1176, 1448], 819, 1086)
        row(15, 'food-sushi food-bun food-milk-tea food-strawberry food-herb food-honey food-grilled-fish',
            [0, 231, 441, 589, 771, 974, 1186, 1448], 0, 248)
        row(15, 'food-carrot food-cake item-potion item-energy food-steak food-fish food-ham food-bone',
            [0, 216, 404, 576, 707, 894, 1085, 1265, 1448], 249, 447)
        row(15, 'chest-wood chest-blue chest-royal item-scroll-blue item-scroll-red item-scroll-green',
            [0, 272, 539, 835, 1034, 1236, 1448], 448, 660)
        row(15, 'weapon-spear weapon-sword weapon-staff weapon-bow weapon-dagger weapon-hammer',
            [0, 258, 486, 716, 975, 1215, 1448], 661, 910)
        row(15, 'gear-helmet gear-armor gear-boots gear-gloves gear-cloak gear-ring gear-necklace gear-collar',
            [0, 184, 399, 580, 736, 976, 1105, 1261, 1448], 911, 1086)
        row(16, 'building-home building-hunt building-portal building-arena',
            [0, 374, 698, 1037, 1448], 0, 293)
        row(16, 'building-shop building-clinic building-warehouse building-market',
            [0, 359, 716, 1073, 1448], 294, 545)
        row(16, 'building-grave building-fountain building-windmill building-training',
            [0, 373, 744, 1081, 1448], 546, 803)
        row(16, 'building-castle building-tower building-boss building-treasury',
            [0, 385, 719, 1094, 1448], 804, 1086)
        row(17, 'rabbit-idle rabbit-happy rabbit-sleep rabbit-eat rabbit-run rabbit-cheer rabbit-fight rabbit-wave',
            [0, 177, 350, 546, 705, 902, 1097, 1273, 1448], 145, 397)
        row(17, 'character-cat character-frog character-slime character-fox character-bird character-dragon character-sheep',
            [0, 190, 390, 571, 793, 1000, 1209, 1448], 398, 632)
        row(17, 'monster-slime monster-mushroom monster-treant monster-bat monster-golem monster-king-slime',
            [0, 189, 379, 601, 825, 1125, 1448], 633, 908)
        row(17, 'face-happy face-angry face-sleep face-love face-confused face-cry face-wink face-sad',
            [0, 191, 387, 563, 740, 907, 1093, 1255, 1448], 909, 1086)
        row(18, 'tree-large tree-medium tree-small pine-large pine-medium pine-small pine-tiny',
            [0, 371, 558, 714, 938, 1119, 1248, 1448], 0, 325)
        row(18, 'bush-white bush-pink bush-small bush-rock flowers-white flowers-pink flowers-blue grass-large grass-small mushrooms-red mushrooms-yellow mushrooms-small',
            [0, 198, 340, 473, 627, 718, 814, 911, 1017, 1108, 1247, 1353, 1448], 326, 454)
        row(18, 'rocks-large rocks-medium rock-small sign-paw sign-arrows fence post-leaf post-plain lantern',
            [0, 245, 391, 514, 661, 826, 1096, 1176, 1286, 1448], 455, 632)
        row(18, 'ground-dirt ground-stone bridge-wood bridge-stone',
            [0, 308, 593, 897, 1323], 633, 790)
        add(sheets[18], 'island-waterfall', (0, 791, 322, 1086))
        add(sheets[18], 'island-small', (323, 791, 532, 1086))
        add(sheets[18], 'island-double-waterfall', (533, 791, 818, 1086))
        add(sheets[18], 'clouds', (819, 791, 1084, 934))
        add(sheets[18], 'treasure', (1085, 791, 1272, 934))
        add(sheets[18], 'campfire', (1273, 772, 1448, 934))
        add(sheets[18], 'ruin-arch', (790, 935, 1028, 1086))
        add(sheets[18], 'ruin-pillar', (1029, 935, 1131, 1086))
        add(sheets[18], 'ruin-stone', (1132, 935, 1234, 1086))
        add(sheets[18], 'crystals-blue', (1235, 935, 1371, 1086))
        add(sheets[18], 'crystal-purple', (1372, 935, 1448, 1086))

    sources = {'ui': args.ui, 'icons': args.icons}
    if args.extras_dir:
        sources.update({str(n): args.extras_dir / f'{n}.png' for n in range(14, 19)})
    source_dir = args.source_dir
    source_dir.mkdir(parents=True, exist_ok=True)
    source_metadata = {}
    for name, source in sources.items():
        destination = source_dir / f'{name}.png'
        if source.resolve() != destination.resolve():
            shutil.copyfile(source, destination)
        source_metadata[name] = {'file': destination.name, 'sha256': hashlib.sha256(source.read_bytes()).hexdigest()}
    manifest = {'sources': source_metadata, 'assets': {}}
    previews = []
    for sheet, name, box, blank in specs:
        image = transparent_background(sheet.crop(box))
        if blank:
            image = blank_button(image)
        assert image.getextrema()[3][0] == 0, name + ': background was not removed'
        assert image.getextrema()[3][1] >= 240, name + ': artwork is empty'
        path = args.out / (name + '.png')
        image.save(path, optimize=True)
        with Image.open(path) as saved:
            assert saved.mode == 'RGBA' and saved.size == image.size
        source_name = 'ui' if sheet is ui else 'icons' if sheet is icons else next(str(n) for n, extra in sheets.items() if sheet is extra)
        manifest['assets'][name] = {'source': source_name, 'crop': box, 'size': image.size, 'blank_center': blank}
        previews.append((name, image))
    (args.out / 'manifest.json').write_bytes((json.dumps(manifest, indent=2) + '\n').encode('utf-8'))
    preview = Image.new('RGB', (960, ((len(previews) + 5) // 6) * 140), '#304b60')
    draw = ImageDraw.Draw(preview)
    for i, (name, image) in enumerate(previews):
        thumb = image.copy()
        thumb.thumbnail((144, 108), Image.Resampling.LANCZOS)
        x, y = (i % 6) * 160, (i // 6) * 140
        preview.paste(thumb, (x + (160 - thumb.width) // 2, y + 4), thumb)
        draw.text((x + 6, y + 118), name, fill='white')
    preview.save(args.out / 'preview.jpg', quality=90)
    total = sum(path.stat().st_size for path in args.out.glob('*.png'))
    print(f'{len(specs)} assets; {total:,} PNG bytes; {args.out}')


if __name__ == '__main__':
    main()
