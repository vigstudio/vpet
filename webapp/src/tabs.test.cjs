const test = require('node:test');
const assert = require('node:assert');
const {TABS, PET_TABS, TAB_KEY, openingTab, readTab, writeTab} = require('./tabs.cjs');

test('the last screen is reopened', () => {
  assert.strictEqual(openingTab('market', 1), 'market');
  assert.strictEqual(openingTab('grave', 0), 'grave', 'the graveyard reads fine without a living pet');
});

test('a tab that is not real falls back to the town', () => {
  for (const bad of [null, undefined, '', 'trade', 'profile', '__proto__', 'CITY', 42]) {
    assert.strictEqual(openingTab(bad, 1), 'city', `accepted ${JSON.stringify(bad)}`);
  }
});

// Burying the last pet used to leave the clinic open, where it crashed on an undefined pet
// and took the whole page with it.
test('pet screens are not reopened when there is no pet', () => {
  for (const tab of PET_TABS) {
    assert.strictEqual(openingTab(tab, 0), 'city', `${tab} opened with no pet`);
    assert.strictEqual(openingTab(tab, 1), tab, `${tab} should open once a pet exists`);
  }
});

test('every tab in the list is a legal starting point', () => {
  for (const tab of TABS) {
    assert.ok(TABS.includes(openingTab(tab, 2)), `${tab} resolved outside the tab list`);
  }
});

test('storage round-trips the tab', () => {
  const store = new Map();
  const fake = {getItem: (k) => store.get(k) ?? null, setItem: (k, v) => store.set(k, v)};
  assert.strictEqual(readTab(fake), null);
  writeTab(fake, 'shop');
  assert.strictEqual(store.get(TAB_KEY), 'shop');
  assert.strictEqual(readTab(fake), 'shop');
});

// Private browsing makes localStorage throw rather than return null; the game must still open.
test('a storage that throws does not break the opening screen', () => {
  const hostile = {
    getItem() { throw new Error('denied'); },
    setItem() { throw new Error('denied'); },
  };
  assert.strictEqual(readTab(hostile), null);
  assert.doesNotThrow(() => writeTab(hostile, 'bag'));
  assert.strictEqual(openingTab(readTab(hostile), 1), 'city');
});
