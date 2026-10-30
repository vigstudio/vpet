const test = require('node:test');
const assert = require('node:assert');
const {beatsOf, sceneLength} = require('./huntscene.cjs');

const MAP = {id: 'rung', name: 'Rừng rậm', emoji: '🌲'};
const label = (id) => `món ${id}`;

test('a plain trip walks in, searches, digs up each find and goes home', () => {
  const b = beatsOf({finds: [{item: 'kiem', rarity: 1}, {item: 'da', rarity: 0}], xp: 12}, MAP, label);
  assert.deepStrictEqual(b.map((x) => x.kind), ['go', 'look', 'find', 'find', 'home']);
  assert.ok(b[2].text.includes('món kiem'));
  assert.ok(sceneLength(b) > 5000);
});

test('an empty-handed trip still says something', () => {
  const b = beatsOf({finds: [], xp: 3}, MAP, label);
  assert.strictEqual(b.filter((x) => x.kind === 'find').length, 0);
  assert.ok(b.some((x) => x.text.includes('chẳng thấy')));
});

test('a wound plays before the walk home', () => {
  const b = beatsOf({finds: [{item: 'da', rarity: 0}], hurt: 17.4, xp: 5}, MAP, label);
  const kinds = b.map((x) => x.kind);
  assert.ok(kinds.indexOf('hurt') < kinds.indexOf('home'));
  assert.ok(b.find((x) => x.kind === 'hurt').text.includes('17'));
});

test('a pet that dies never walks home', () => {
  const b = beatsOf({finds: [], hurt: 60, note: '💀 Mun đã bỏ mạng…'}, MAP, label);
  assert.ok(b.some((x) => x.kind === 'die'));
  assert.ok(!b.some((x) => x.kind === 'home'));
});

test('every beat has a position on the scene and a duration', () => {
  const b = beatsOf({finds: [{item: 'a', rarity: 3}, {item: 'b', rarity: 2}, {item: 'c', rarity: 0}], xp: 9, hurt: 4}, MAP, label);
  assert.ok(b.every((x) => x.x >= 0 && x.x <= 100 && x.ms > 0 && x.text));
});
