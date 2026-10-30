# VPet RPG UI assets

Source: **Free Basic Pixel Art UI for RPG** by CraftPix  
Archive: `craftpix-net-255216-free-basic-pixel-art-ui-for-rpg.zip`  
License: [CraftPix Freebies License](https://craftpix.net/file-licenses/)

The checked-in PNGs are small derived crops used as finished UI chrome:

- `rpg-panel.png`: resizable parchment/window frame from `PNG/Inventory.png`.
- `rpg-button.png`: four blank button states from `PNG/Buttons.png`; labels remain accessible HTML.
- `rpg-slot.png`: inventory cell from `PNG/Inventory.png`.
- `rpg-inventory.png`, `rpg-equipment.png`, `rpg-shop.png`, `rpg-craft.png`, `rpg-settings.png`, `rpg-result.png`: cropped screen motifs used as visible, scalable chrome on their matching VPet screens.
- `rpg-icons.png`: the pack's compact icon sheet, used for distinct Divine gear artwork instead of emoji or lower-rank icon reuse.

Regenerate from the licensed archive without extracting it into the repository:

```bash
python3 -I webapp/assets/make-rpg-ui.py /path/to/craftpix-net-255216-free-basic-pixel-art-ui-for-rpg.zip
```

The original ZIP, PSDs, full PNG sheets, preview text, coupon and font are intentionally excluded. Do not redistribute source art or expose it as a reusable asset pack. The license prohibits using the assets for AI/ML training or testing.
