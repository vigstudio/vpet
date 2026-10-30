const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const assets = path.resolve(__dirname, '../assets');
const expected = {
  'rpg-panel.png': [98, 101],
  'rpg-button.png': [128, 12],
  'rpg-slot.png': [18, 18],
  'rpg-inventory.png': [98, 101],
  'rpg-equipment.png': [146, 101],
  'rpg-shop.png': [98, 149],
  'rpg-craft.png': [200, 160],
  'rpg-settings.png': [98, 149],
  'rpg-result.png': [96, 88],
  'rpg-icons.png': [96, 304],
};

const size = (file) => {
  const data = fs.readFileSync(file);
  assert.equal(data.subarray(1, 4).toString(), 'PNG');
  return [data.readUInt32BE(16), data.readUInt32BE(20)];
};

test('derived RPG chrome has the expected compact frames', () => {
  for (const [name, dimensions] of Object.entries(expected)) {
    assert.deepEqual(size(path.join(assets, name)), dimensions, name);
  }
});

test('Divine gear has eight unique atlas entries', () => {
  const source = fs.readFileSync(path.join(assets, 'items.js'), 'utf8');
  const divine = ['kiemthanlong', 'giapthanlong', 'muthiende', 'gangthienloi', 'hiathienvuong', 'nhanthienmenh', 'khienthienha', 'aothienmenh'];
  for (const id of divine) {
    assert.equal((source.match(new RegExp(`(?:^|,)${id}(?:,|'\\.split)`,'g')) || []).length, 1, id);
    assert.deepEqual(size(path.join(assets, 'divine', `${id}.png`)), [32, 32], id);
  }
  assert.deepEqual(size(path.join(assets, 'items.png')), [3008, 32]);
});

test('source CraftPix sheets are not shipped with the plugin', () => {
  const forbidden = /\.(?:zip|psd|pdf)$/i;
  const sourceSheet = /^(?:Inventory|Buttons|Main_tiles|Shop|Craft|Settings|Win_loose)\.png$/i;
  for (const name of fs.readdirSync(assets)) {
    assert.equal(forbidden.test(name) || sourceSheet.test(name), false, name);
  }
});
