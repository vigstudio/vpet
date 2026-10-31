"""Compose local user-supplied artwork. Run with Python + Pillow."""
from pathlib import Path
from collections import deque
import random

from PIL import Image, ImageFilter

BASE = Path(__file__).parent
ART = BASE / 'fantasy'
LIB = BASE / 'library'


def stamp(canvas, name, box, library=False):
    image = Image.open((LIB if library else ART) / (name + '.png')).convert('RGBA')
    if name in {'island-large', 'island-small', 'clouds'}:
        mask = image.getchannel('A').point(lambda a: 255 if a > 100 else 0)
        pixels, seen, components = mask.load(), set(), []
        for y in range(image.height):
            for x in range(image.width):
                if not pixels[x, y] or (x, y) in seen:
                    continue
                queue, component = deque([(x, y)]), []
                seen.add((x, y))
                while queue:
                    xx, yy = queue.popleft()
                    component.append((xx, yy))
                    for nx, ny in ((xx-1, yy), (xx+1, yy), (xx, yy-1), (xx, yy+1)):
                        if 0 <= nx < image.width and 0 <= ny < image.height and pixels[nx, ny] and (nx, ny) not in seen:
                            seen.add((nx, ny))
                            queue.append((nx, ny))
                components.append(component)
        keep = max(components, key=len)
        clean = Image.new('L', image.size)
        for xy in keep:
            clean.putpixel(xy, image.getpixel(xy)[3])
        image.putalpha(clean)
        image = image.crop(image.getbbox())
    if name == 'island-large':
        split = round(image.height * .54)
        top = image.crop((0, 0, image.width, split)).resize((box[2], round(box[3] * .72)), Image.Resampling.LANCZOS)
        bottom = image.crop((0, split, image.width, image.height)).resize((box[2], box[3] - top.height), Image.Resampling.LANCZOS)
        image = Image.new('RGBA', box[2:])
        image.alpha_composite(top)
        image.alpha_composite(bottom, (0, top.height))
        image = image.filter(ImageFilter.GaussianBlur(.7))
    else:
        image = image.resize(box[2:], Image.Resampling.LANCZOS)
    canvas.alpha_composite(image, box[:2])


def main():
    sky = Image.open(LIB / 'scene-sky.png').convert('RGBA').resize((1600, 900), Image.Resampling.LANCZOS)
    sky = sky.filter(ImageFilter.GaussianBlur(3))
    for x, y, w in [(30, 120, 300), (1220, 110, 270), (1050, 620, 360), (-100, 660, 450)]:
        stamp(sky, 'clouds', (x, y, w, w // 2))
    stamp(sky, 'island-small', (1250, 120, 220, 190))
    stamp(sky, 'building-castle', (1290, 60, 130, 110))
    stamp(sky, 'island-large', (70, 200, 1460, 650), library=True)
    # Paths share one plateau; buildings stay DOM controls rather than baked UI.
    for x, y, w, h in [(200, 320, 350, 155), (540, 310, 360, 170),
                       (870, 310, 340, 175), (310, 440, 360, 160),
                       (650, 440, 360, 160), (975, 430, 340, 160)]:
        stamp(sky, 'ground-stone', (x, y, w, h))
    trees = [(110, 250, 140), (280, 210, 140), (500, 190, 110),
             (950, 190, 125), (1260, 240, 130), (1370, 340, 95),
             (150, 440, 100), (1240, 480, 115)]
    for i, (x, y, w) in enumerate(trees):
        stamp(sky, 'tree-medium' if i % 2 else 'pine-medium', (x, y, w, int(w * 1.25)))
    rng = random.Random(21)
    for _ in range(28):
        x, y = rng.randint(170, 1320), rng.choice([310, 575, 595])
        stamp(sky, rng.choice(['flowers-pink', 'flowers-white', 'bush-small']), (x, y, 45, 27))
    stamp(sky, 'building-fountain', (720, 455, 140, 105))
    stamp(sky, 'bridge-stone', (1050, 595, 240, 115))
    stamp(sky, 'lantern', (1280, 510, 65, 120))
    sky.convert('RGB').save(BASE / 'fantasy-city.webp', quality=88, method=6)

    meadow = Image.open(LIB / 'scene-meadow.png').convert('RGBA').resize((1000, 700), Image.Resampling.LANCZOS)
    meadow = meadow.filter(ImageFilter.GaussianBlur(2))
    stamp(meadow, 'tree-large', (-90, 100, 360, 420))
    stamp(meadow, 'tree-medium', (770, 160, 290, 340))
    stamp(meadow, 'ground-dirt', (180, 430, 650, 250))
    stamp(meadow, 'flowers-pink', (40, 565, 190, 100))
    stamp(meadow, 'bush-white', (760, 540, 250, 140))
    meadow.convert('RGB').save(BASE / 'fantasy-pet.webp', quality=88, method=6)
    for name, size in [('fantasy-city', (1600, 900)), ('fantasy-pet', (1000, 700))]:
        with Image.open(BASE / (name + '.webp')) as image:
            assert image.size == size
        print(name, (BASE / (name + '.webp')).stat().st_size)


if __name__ == '__main__':
    main()
