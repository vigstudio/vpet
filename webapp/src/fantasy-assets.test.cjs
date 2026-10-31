const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const points = require('../assets/realm-points.json');

test('the fantasy town keeps nine distinct destinations with non-overlapping controls', () => {
  assert.deepEqual(points.map(p => p.id).sort(), ['pet','hunt','world','shop','bag','market','duel','clinic','grave'].sort());
  for (const a of points) {
    assert.ok(a.x >= 8 && a.x <= 92 && a.y >= 12 && a.y <= 88, a.id);
    for (const b of points) {
      if (a === b) continue;
      assert.ok(Math.abs(a.x - b.x) >= 16 || Math.abs(a.y - b.y) >= 24, `${a.id}/${b.id}`);
    }
  }
});

test('every cut library asset is a PNG with its recorded dimensions', () => {
  const root = path.join(__dirname, '../assets/library');
  const manifest = JSON.parse(fs.readFileSync(path.join(root, 'manifest.json'), 'utf8'));
  assert.equal(Object.keys(manifest.assets).length, 98);
  for (const [name, entry] of Object.entries(manifest.assets)) {
    const png = fs.readFileSync(path.join(root, `${name}.png`));
    assert.equal(png.subarray(1, 4).toString(), 'PNG', name);
    assert.deepEqual([png.readUInt32BE(16), png.readUInt32BE(20)], entry.size, name);
  }
});
