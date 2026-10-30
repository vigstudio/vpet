// Which screen the game opens on. Kept out of Panel.jsx so the rules can be tested without
// a browser: a remembered tab must never open a screen that cannot render.

const TABS = ['city', 'pet', 'quest', 'world', 'hunt', 'bag', 'shop', 'clinic', 'market', 'duel', 'grave'];
const PHONE_TABS = 4; // the fifth nav slot is "More"
const TAB_KEY = 'vpet.tab';

// Screens built around the selected pet. With no pet they have nothing to show, so the town
// takes over rather than opening an empty page.
const PET_TABS = ['pet', 'hunt', 'world', 'duel', 'clinic'];

// openingTab picks the starting screen. Anything unrecognised — an older build's tab, a
// corrupted value, a tab that needs a pet the player no longer has — falls back to the town.
function openingTab(saved, pets = 0) {
  if (!TABS.includes(saved)) {
    return 'city';
  }
  if (!pets && PET_TABS.includes(saved)) {
    return 'city';
  }
  return saved;
}

function readTab(storage) {
  try {
    return storage.getItem(TAB_KEY);
  } catch (e) {
    return null; // private mode
  }
}

function writeTab(storage, tab) {
  try {
    storage.setItem(TAB_KEY, tab);
  } catch (e) {
    // private mode: the tab simply is not remembered
  }
}

module.exports = {TABS, PHONE_TABS, PET_TABS, TAB_KEY, openingTab, readTab, writeTab};
