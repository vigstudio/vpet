# Scenic realm artwork

- Source models: [Kenney Fantasy Town Kit 2.0](https://kenney.nl/assets/fantasy-town-kit).
- Download: https://kenney.nl/media/pages/assets/fantasy-town-kit/efe948d309-1754222374/kenney_fantasy-town-kit_2.0.zip
- License: **CC0 1.0**. Original notice: `FANTASY-TOWN-LICENSE.txt`.
- `make-realm.py` composes original floating islands, paths and modular landmarks, reads OBJ geometry + atlas only, renders six optimized WebP runtime scenes using Pillow. No downloaded code is executed. Models themselves are not shipped to browsers.
- Rebuild: `python3 -I webapp/assets/make-realm.py /path/to/kenney_fantasy-town-kit_2.0.zip` from plugin root, with Pillow available. Extracting the archive is unnecessary.
- `Realm.jsx` imports WebP to keep the plugin bundle small. `realm-points.json` is generated with the same projection as city art; DOM buttons use its percentages.
- The supplied Gunny screenshot is a visual reference only. No Gunny artwork, logos, player data or currency display is included.
