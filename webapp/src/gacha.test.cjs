const test = require('node:test');
const assert = require('node:assert');

const {ACTS, CELLS, actAt, buildReel, totalMs} = require('./gacha.cjs');

const prizes = [
  {id: 'a', rarity: 0}, {id: 'b', rarity: 0}, {id: 'c', rarity: 1},
  {id: 'd', rarity: 2}, {id: 'e', rarity: 3},
];

test('the prize the server picked is the cell that stops under the needle', () => {
  const won = {id: 'c', rarity: 1};
  const reel = buildReel(prizes, won, 0);
  assert.strictEqual(reel.length, CELLS + 1);
  assert.strictEqual(reel[reel.length - 1].id, 'c');
});

test('the cell before the prize is rarer, so the crawl passes a better item', () => {
  const won = {id: 'c', rarity: 1};
  const reel = buildReel(prizes, won, 0);
  assert.ok((reel[reel.length - 2].rarity || 0) > won.rarity);
});

test('winning the rarest item leaves no tease to fake', () => {
  const won = {id: 'e', rarity: 3};
  const reel = buildReel(prizes, won, 0);
  assert.strictEqual(reel[reel.length - 1].id, 'e');
  assert.strictEqual(reel.length, CELLS + 1);
});

test('two spins in a row do not scroll the same filler', () => {
  const won = {id: 'c', rarity: 1};
  const first = buildReel(prizes, won, 0).map((it) => it.id).join('');
  const second = buildReel(prizes, won, 1).map((it) => it.id).join('');
  assert.notStrictEqual(first, second);
});

test('nothing to spin yields no reel rather than throwing', () => {
  assert.deepStrictEqual(buildReel([], {id: 'c'}, 0), []);
  assert.deepStrictEqual(buildReel(prizes, null, 0), []);
});

test('the acts run in order and end on the slow crawl', () => {
  assert.strictEqual(actAt(0), 'launch');
  assert.strictEqual(actAt(ACTS[0].ms + 10), 'blur');
  assert.strictEqual(actAt(totalMs() - 10), 'creep');
  assert.strictEqual(actAt(totalMs()), 'done');
});

test('the creep is the longest act: the last cells are what the player waits on', () => {
  const creep = ACTS[ACTS.length - 1].ms;
  assert.ok(ACTS.every((act) => act.ms <= creep));
});
