# Item asset sources

All listed artwork is CC0/public domain. Derived Divine icons remain CC0.

## Existing atlas

- Dungeon Crawl Stone Soup tiles, CC0: https://opengameart.org/content/dungeon-crawl-32x32-tiles
- Used by the standard, Mythic, consumable, gem, card, shield and cloak icons in `items.png`.

## Divine gear

- Idylwild's Armory, CC0: https://opengameart.org/content/idylwilds-armory
  - `giapthanlong` ← platemail
  - `muthiende` ← platemail helmet
  - `gangthienloi` ← gauntlets; cyan lightning highlight added
  - `hiathienvuong` ← greaves
  - `nhanthienmenh` ← gold ring
  - `khienthienha` ← tower shield
  - `aothienmenh` ← cloak
- RPG Fantasy Icon Set, CC0: https://opengameart.org/content/rpg-fantasy-icon-set
  - `kiemthanlong` ← sword

The selected source PNG files are input data only. `make-divine-items.py` scales them with nearest-neighbour sampling and recolors them to crimson–gold. It writes eight deterministic 32×32 files under `divine/`.

Regenerate from the plugin root:

```bash
python3 -I webapp/assets/make-divine-items.py /path/to/selected-sources webapp/assets/divine
python3 webapp/assets/make-items.py
```

Expected source filenames: `sword.png`, `platemail.png`, `platemail_helmet.png`, `gauntlets.png`, `greaves.png`, `gold_ring.png`, `tower_shield.png`, `cloak.png`.
