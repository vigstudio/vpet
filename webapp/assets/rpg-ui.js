import panel from './rpg-panel.png';
import buttons from './rpg-button.png';
import slot from './rpg-slot.png';
import inventory from './rpg-inventory.png';
import equipment from './rpg-equipment.png';
import shop from './rpg-shop.png';
import craft from './rpg-craft.png';
import settings from './rpg-settings.png';
import result from './rpg-result.png';
import icons from './rpg-icons.png';

export const RPG_UI = Object.freeze({panel, buttons, slot, inventory, equipment, shop, craft, settings, result, icons});
export const RPG_UI_META = Object.freeze({
  panel: {width: 98, height: 101},
  buttons: {width: 128, height: 12, states: 4},
  slot: {width: 18, height: 18},
  inventory: {width: 98, height: 101},
  equipment: {width: 146, height: 101},
  shop: {width: 98, height: 149},
  craft: {width: 200, height: 160},
  settings: {width: 98, height: 149},
  result: {width: 96, height: 88},
  icons: {width: 96, height: 304},
});
