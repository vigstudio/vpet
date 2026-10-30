# Packs the item icons into one sprite sheet the webapp inlines.
# Art: Dungeon Crawl Stone Soup 32x32 tiles, CC0 (public domain), via opengameart.org.
# Run from the plugin root after downloading the pack to /tmp/icons_dl/picked:
#   python3 webapp/assets/make-items.py
from pathlib import Path

from PIL import Image

ASSETS = Path(__file__).resolve().parent
SRC = '/tmp/icons_dl/picked/%s.png'
FOOD = '/tmp/icons_dl/dcss/full/item/food/%s.png'
S = 32

# item id -> icon, in the order the sheet stores them
ICONS = [
    # weapons: rusty knife -> sword -> trident -> wizard staff
    ('dao', 'knife'), ('kiem', 'short_sword'), ('dinhba', 'trident_1'), ('gaythan', 'staff_1'),
    # body armour
    ('aovai', 'robe_1'), ('giapda', 'leather_armor_1'), ('giapthep', 'plate_mail_1'), ('longbao', 'robe_art_2'),
    # helmets
    ('nonre', 'hat_1'), ('muda', 'elven_leather_helm'), ('muthep', 'helmet_1_visored'), ('vuongmien', 'helmet_art_1'),
    # gloves
    ('gangvai', 'glove_1'), ('gangda', 'glove_2'), ('gangthep', 'gauntlet_1'), ('gangrong', 'glove_4_gauntlets'),
    # boots
    ('depco', 'low_boots'), ('giayda', 'boots_brown'), ('giaythep', 'boots_iron'), ('hiabay', 'boots_stripe'),
    # rings and trinkets
    ('nhandong', 'ring_gold'), ('chuoingoc', 'amu_celtic_blue'), ('nhanbac', 'ring_sapphire'), ('nhanrong', 'ring_opal'),
    # loot and consumables
    ('ngoc', 'misc_crystal'), ('da', 'misc_stone'), ('thuoc', 'potion_ruby'), ('vitamin', 'potion_emerald'),
    ('thaomoc', 'potion_clear'), ('long', 'misc_rune'), ('sach', 'scroll'),
    ('the_skin', 'scroll_blue'), ('the_slot', 'key'), ('the_nghi', 'magic_lamp'),
    ('tui', 'chest_2_closed'), ('tui_y', 'chest_2_open'), ('tui_vukhi', 'chest'), ('tui_vang', 'gold_pile_25'),
]

# 1.14: the shield and cloak slots, the mythic rung and the new cards, all from the full pack
EXTRA = '/tmp/icons_dl/dcss/full/item/%s.png'
EXTRA_ICONS = [
    # shields, scrap to silver
    ('motnap', 'armor/shields/buckler_1_old'), ('khiengo', 'armor/shields/buckler_2_new'),
    ('khiensat', 'armor/shields/shield_1'), ('khienbac', 'armor/shields/shield_2_kite'),
    # cloaks, scrap to starfall
    ('aotoi', 'armor/back/cloak_1_leather'), ('aogam', 'armor/back/cloak_2'),
    ('aolong', 'armor/back/cloak_3'), ('aosao', 'armor/back/cloak_4'),
    # the mythic rung: forged only, so it gets the pack's showiest art
    ('kiemrong', 'weapon/demon_blade'), ('giaprong', 'armor/torso/gold_dragon_armor_new'),
    ('muthan', 'armor/headgear/helmet_art_2'), ('gangthan', 'armor/hands/glove_5'),
    ('hiathan', 'armor/feet/boots_2_jackboots'), ('nhanthan', 'ring/opal'),
    ('khienthan', 'armor/shields/shield_of_reflection'), ('aothan', 'armor/back/cloak_4'),
    # new cards, food and restoratives
    ('the_ten', 'scroll/scroll-yellow'), ('the_quen', 'scroll/scroll-grey'),
    ('banhtet', 'food/sausage'), ('chegiai', 'food/grape'), ('nuocmia', 'food/lychee_new'),
    ('chaogiai', 'potion/cloudy'), ('tinhdau', 'potion/murky'), ('sammy', 'potion/pink'),
    ('mu_bao', 'armor/headgear/helmet_1'), ('vongtay', 'amulet/celtic_yellow'),
]

# food and trinkets come from the pack's food folder, which the curated copy left out
FOOD_ICONS = [
    ('com', 'bread_ration_new'), ('banhmi', 'bread_ration_old'), ('pho', 'meat_ration_new'),
    ('sushi', 'beef_jerky_new'), ('banhkem', 'lump_of_royal_jelly_new'), ('trasua', 'honeycomb_new'),
    ('ot', 'strawberry_new'), ('micay', 'pizza_new'), ('saurieng', 'snozzcumber'),
    ('mamtom', 'chunk'), ('trungthoi', 'chunk_rotten'), ('nam', 'choko'),
    ('caphe', 'piece_of_ambrosia_new'), ('mathu', 'honeycomb_old'),
]

# restoratives and accessories, straight from the full pack
FULL = '/tmp/icons_dl/dcss/full/item/%s.png'
FULL_ICONS = [
    ('banggac', 'misc/misc_box'), ('thuocbo', 'potion/orange_new'), ('canhtien', 'potion/potion_golden'),
    ('nuocthanh', 'misc/misc_phial'), ('nonla', 'armor/headgear/hat_2'), ('kinh', 'misc/mirror'),
    ('no', 'armor/back/cloak_4'), ('khan', 'armor/back/cloak_3'),
]

DIVINE = str(ASSETS / 'divine' / '%s.png')
DIVINE_ICONS = [
    ('kiemthanlong', 'kiemthanlong'), ('giapthanlong', 'giapthanlong'),
    ('muthiende', 'muthiende'), ('gangthienloi', 'gangthienloi'),
    ('hiathienvuong', 'hiathienvuong'), ('nhanthienmenh', 'nhanthienmenh'),
    ('khienthienha', 'khienthienha'), ('aothienmenh', 'aothienmenh'),
]


def main():
    all_icons = ([(i, SRC % n) for i, n in ICONS] + [(i, FOOD % n) for i, n in FOOD_ICONS]
                 + [(i, FULL % n) for i, n in FULL_ICONS] + [(i, EXTRA % n) for i, n in EXTRA_ICONS]
                 + [(i, DIVINE % n) for i, n in DIVINE_ICONS])
    sheet = Image.new('RGBA', (len(all_icons) * S, S))
    for i, (_, path) in enumerate(all_icons):
        im = Image.open(path).convert('RGBA')
        if im.size != (S, S):
            im = im.resize((S, S), Image.NEAREST)
        sheet.paste(im, (i * S, 0), im)
    sheet.save('webapp/assets/items.png')
    ids = ','.join(i for i, _ in all_icons)
    with open('webapp/assets/items.js', 'w', newline='\n') as f:
        f.write('// Generated by make-items.py. CC0 sources: Dungeon Crawl Stone Soup, Idylwild Armory, RPG Fantasy Icon Set.\n')
        f.write("export const ITEM_ICONS = '%s'.split(',');\n" % ids)
    print(sheet.size, len(all_icons))


main()
