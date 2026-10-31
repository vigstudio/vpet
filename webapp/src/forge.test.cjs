const assert = require('node:assert/strict');
const test = require('node:test');

const {benchItems, benchActions} = require('./forge.cjs');

const catalog = [
  {id: 'kiem', name: 'Kiếm sắt', rarity: 1, kind: 'gear'},
  {id: 'dinhba', name: 'Đinh ba', rarity: 2, kind: 'gear'},
  {id: 'giapda', name: 'Giáp da', rarity: 1, kind: 'gear'},
  {id: 'com', name: 'Cơm', rarity: 0, kind: 'food'},
];
const gearStats = {kiem: {slot: 'vukhi'}, dinhba: {slot: 'vukhi'}, giapda: {slot: 'giap'}};

test('the bench lists worn gear first, then the bag, and never food', () => {
  const rows = benchItems({
    worn: [{slot: 'vukhi', item: 'kiem', level: 3}],
    inv: {dinhba: 2, com: 9},
    enhanced: [{item: 'giapda', level: 1, count: 1}],
    gearStats, catalog,
  });
  assert.deepEqual(rows.map((r) => r.key), ['worn:vukhi', 'bag:dinhba:0', 'bag:giapda:1']);
  assert.equal(rows[0].worn, true);
  assert.equal(rows[0].level, 3);
  assert.equal(rows.some((r) => r.item.id === 'com'), false, 'food is not gear');
});

test('a +0 and an enhanced copy of one item are separate bench rows', () => {
  const rows = benchItems({inv: {kiem: 2}, enhanced: [{item: 'kiem', level: 4, count: 1}], gearStats, catalog});
  assert.equal(rows.length, 2);
  assert.deepEqual(rows.map((r) => r.level).sort(), [0, 4]);
});

test('an empty bag produces an empty bench rather than throwing', () => {
  assert.deepEqual(benchItems({}), []);
});

const wornRow = {key: 'worn:vukhi', item: catalog[0], level: 3, count: 1, slot: 'vukhi', worn: true};
const bagRow = {key: 'bag:kiem:0', item: catalog[0], level: 0, count: 3, slot: 'vukhi', worn: false};
const enhance = [{slot: 'vukhi', item: 'kiem', level: 3, max: 10, coin: 80, shards: 2,
  current: {atk: 5.6}, next: {atk: 5.8}, affordable: true}];
const sockets = [{slot: 'vukhi', item: 'kiem', gems: ['gem_atk_1'], free: 2}];
const recipes = [{from: 'kiem', to: 'dinhba', have: 3, need: 3, cost: 80, ready: true}];

test('worn gear offers enhance, socket and fuse with their real costs', () => {
  const acts = benchActions(wornRow, {enhance, sockets, recipes, coin: 1000, shards: 50, catalog});
  const byKind = Object.fromEntries(acts.map((a) => [a.kind, a]));
  assert.equal(byKind.enhance.blocked, '');
  assert.equal(byKind.enhance.coin, 80);
  assert.deepEqual([byKind.enhance.level, byKind.enhance.next.atk], [3, 5.8], 'shows where it is and what it becomes');
  assert.equal(byKind.socket.free, 2);
  assert.equal(byKind.socket.blocked, '');
});

test('bag gear explains that it must be equipped, instead of hiding the action', () => {
  const acts = benchActions(bagRow, {enhance, sockets, recipes, coin: 1000, shards: 50, catalog});
  const byKind = Object.fromEntries(acts.map((a) => [a.kind, a]));
  assert.equal(byKind.enhance.blocked, 'equip');
  assert.equal(byKind.socket.blocked, 'equip');
  assert.equal(byKind.fuse.blocked, '', 'a +0 bag copy is exactly what fusing wants');
});

test('a short purse names the shortfall rather than only disabling the button', () => {
  const poor = benchActions(wornRow, {
    enhance: [{...enhance[0], affordable: false}], sockets, recipes, coin: 30, shards: 1, catalog,
  });
  const act = poor.find((a) => a.kind === 'enhance');
  assert.equal(act.blocked, 'cost');
  assert.equal(act.shortCoin, 50);
  assert.equal(act.shortShards, 1);
});

test('enhanced gear cannot be fused: the level would be lost to the payload', () => {
  const row = {...bagRow, level: 4, key: 'bag:kiem:4'};
  const acts = benchActions(row, {enhance, sockets, recipes, coin: 1000, shards: 50, catalog});
  const byKind = Object.fromEntries(acts.map((a) => [a.kind, a]));
  assert.equal(byKind.fuse.blocked, 'enhanced');
  assert.ok(byKind.restore, 'restoring to +0 is the way out, so it is offered');
  assert.equal(byKind.restore.level, 4);
});

test('the top of a ladder says so instead of offering an impossible fuse', () => {
  const acts = benchActions(bagRow, {enhance, sockets, recipes: [], coin: 1000, shards: 50, catalog});
  assert.equal(acts.find((a) => a.kind === 'fuse').blocked, 'top');
});

test('missing ingredients report how many are missing', () => {
  const acts = benchActions(bagRow, {
    enhance, sockets, coin: 1000, shards: 50, catalog,
    recipes: [{from: 'kiem', to: 'dinhba', have: 1, need: 3, cost: 80, ready: false}],
  });
  const fuse = acts.find((a) => a.kind === 'fuse');
  assert.equal(fuse.blocked, 'parts');
  assert.equal(fuse.shortParts, 2);
  assert.equal(fuse.into.name, 'Đinh ba', 'the player sees what they are working towards');
});

test('a maxed piece and a full socket row both say so', () => {
  const maxed = benchActions(wornRow, {
    enhance: [{...enhance[0], level: 10, max: 10}],
    sockets: [{slot: 'vukhi', item: 'kiem', gems: ['a', 'b', 'c'], free: 0}],
    recipes, coin: 1000, shards: 50, catalog,
  });
  assert.equal(maxed.find((a) => a.kind === 'enhance').blocked, 'max');
  assert.equal(maxed.find((a) => a.kind === 'socket').blocked, 'full');
});

test('no selection yields no actions', () => {
  assert.deepEqual(benchActions(null, {}), []);
});

test('top tier explains missing material rather than missing gold', () => {
  const recipe = {...recipes[0], ready: false, material: 'tinhhon', material_have: 2, material_need: 3};
  const result = benchActions(bagRow, {recipes: [recipe], coin: 99999, catalog}).find((a) => a.kind === 'fuse');
  assert.equal(result.blocked, 'material');
});
