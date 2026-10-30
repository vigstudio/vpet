# Builds the town art from Kenney's CC0 "Tiny Town" tileset: one 16px tile grid for the
# ground and one sprite sheet with the eight buildings, hue-shifted per feature.
#
# Only needed when the town layout changes. Grab the tiles first, then run from the plugin root:
#   curl -sLo /tmp/tt.zip https://kenney.nl/media/pages/assets/tiny-town/a415fbeb49-1735736916/kenney_tiny-town.zip
#   mkdir -p /tmp/kenney_dl/tt && (cd /tmp/kenney_dl/tt && unzip -oq /tmp/tt.zip)
#   python3 webapp/assets/make-city.py
# Kenney assets are CC0: free for commercial use, credit optional.
import colorsys
from PIL import Image, ImageDraw, ImageFilter

SRC = '/tmp/kenney_dl/tt/Tiles/tile_%04d.png'
S = 16
t = lambda n: Image.open(SRC % n).convert('RGBA')

def shift(im, deg, sat=1.0):
    """Rotate hue so one red roof can serve eight different buildings."""
    out = im.copy()
    px = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = px[x, y]
            if a == 0:
                continue
            h, l, s = colorsys.rgb_to_hls(r / 255, g / 255, b / 255)
            h = (h + deg / 360.0) % 1.0
            r, g, b = colorsys.hls_to_rgb(h, l, min(1.0, s * sat))
            px[x, y] = (int(r * 255), int(g * 255), int(b * 255), a)
    return out

# --- ground: 30x19 tiles of grass, a dirt lane and some trees ---
W, H = 30, 19
ground = Image.new('RGBA', (W * S, H * S))
grass = [t(0), t(1), t(2)]
for y in range(H):
    for x in range(W):
        g = grass[(x * 7 + y * 3) % 3 if (x + y) % 5 == 0 else 0]
        ground.paste(g, (x * S, y * S))

# the lane: drawn as a smooth wide stroke, then filled with the dirt tile through a mask,
# so the road curves instead of climbing in steps
road = Image.new('L', ground.size, 0)
rd = ImageDraw.Draw(road)
for pts in ([(0, 150), (90, 165), (180, 140), (260, 150), (350, 120), (430, 95), (480, 80)],
            [(60, 235), (150, 215), (250, 240), (340, 218), (480, 225)]):
    rd.line(pts, fill=255, width=26, joint='curve')
road = road.filter(ImageFilter.MaxFilter(3))
dirt_fill = Image.new('RGBA', ground.size)
dirt = t(13)
for y in range(0, ground.height, S):
    for x in range(0, ground.width, S):
        dirt_fill.paste(dirt, (x, y))
ground.paste(dirt_fill, (0, 0), road)

# trees are two tiles tall: crown on top, trunk underneath, and never on the road
TREES = [(1, 4), (4, 2), (8, 4), (16, 2), (22, 10), (27, 10), (2, 16), (12, 17), (21, 17),
         (28, 16), (6, 11), (19, 10), (25, 17), (13, 4), (9, 10), (24, 7)]
for i, (x, y) in enumerate(TREES):
    top, trunk = [(3, 15), (4, 16), (5, 17)][i % 3]
    box = road.crop((x * S, y * S, x * S + S, y * S + 2 * S))
    if box.getextrema()[1] > 0:
        continue
    for n, yy in ((top, y), (trunk, y + 1)):
        im = t(n)
        ground.paste(im, (x * S, yy * S), im)
for x, y in [(10, 3), (14, 11), (24, 16), (5, 6)]:
    m = t(29)
    ground.paste(m, (x * S, y * S), m)
ground.save('webapp/assets/city-ground.png')

# --- buildings: three tiles wide, hue-shifted roof per feature ---
# roof row, wall row and the door, taken from the tileset's house block
ROOF_L, ROOF_M, ROOF_R, GABLE = 52, 53, 54, 67
WALL_A, WALL_B, DOOR = 72, 73, 85
HUE = [0, 100, -140, 30, -25, 220, -10, 185]   # one per house, in HOUSES order
SAT = [1, .9, .9, 1.05, .6, .9, 1, .35]

sheet = Image.new('RGBA', (len(HUE) * 3 * S, 3 * S))
for i, deg in enumerate(HUE):
    ox = i * 3 * S
    roof = [shift(t(n), deg, SAT[i]) for n in (ROOF_L, GABLE, ROOF_R)]
    for c, im in enumerate(roof):
        sheet.paste(im, (ox + c * S, 0), im)
    for c, n in enumerate((WALL_A, WALL_B, WALL_A)):
        im = t(n)
        sheet.paste(im, (ox + c * S, S), im)
    door = t(DOOR)
    sheet.paste(t(WALL_B), (ox, 2 * S))
    sheet.paste(door, (ox + S, 2 * S), door)
    sheet.paste(t(WALL_A), (ox + 2 * S, 2 * S))
    # bottom row stays transparent so the building sits on whatever ground is behind it
sheet.save('webapp/assets/city-houses.png')
print(ground.size, sheet.size)
