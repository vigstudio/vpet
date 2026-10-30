const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const {PARTS, buildGrid, randomParts, W, H} = require('./sprite.cjs');

test('part counts match the server', () => {
  const go = fs.readFileSync(path.join(__dirname, '../../internal/game/game.go'), 'utf8');
  const block = go.match(/partLimits = map\[string\]int\{([^}]*)\}/)[1];
  const server = Object.fromEntries([...block.matchAll(/"(\w+)": (\d+)/g)].map((m) => [m[1], Number(m[2])]));
  const client = Object.fromEntries(Object.entries(PARTS).map(([k, v]) => [k, v.length]));
  assert.deepStrictEqual(client, server);
});

test('every part option renders inside the grid', () => {
  const effects = ['', 'cay', 'hoi', 'ngon', 'hyper', 'ngodoc', 'dau', 'cuoi', 'ngu'];
  for (const [part, options] of Object.entries(PARTS)) {
    options.forEach((_, i) => {
      for (const effect of effects) {
        const grid = buildGrid({...randomParts(() => 0.3), [part]: i}, {effect, frame: i, sick: i % 2 === 0, blink: i === 1});
        assert.strictEqual(grid.length, W * H);
        assert.ok(grid.filter(Boolean).length > 80, `${part}=${i} ${effect} draws something`);
      }
    });
  }
  assert.ok(buildGrid({}, {dead: true}).filter(Boolean).length > 100);
});

test('parts change the picture', () => {
  const base = randomParts(() => 0);
  const seen = new Set();
  PARTS.head.forEach((_, i) => seen.add(buildGrid({...base, head: i}, {still: true}).join()));
  assert.strictEqual(seen.size, PARTS.head.length);
});

test('every server accessory is drawn', () => {
  const go = fs.readFileSync(path.join(__dirname, '../../internal/game/game.go'), 'utf8');
  const accs = [...go.matchAll(/ID: "(\w+)",[^\n]*Kind: "acc"/g)].map((m) => m[1]);
  assert.ok(accs.length >= 4);
  const base = randomParts(() => 0.5);
  const plain = buildGrid(base, {still: true}).join();
  accs.forEach((acc) => assert.notStrictEqual(buildGrid(base, {still: true, acc}).join(), plain, acc));
});

test('nothing is drawn against the top edge (tall parts fit)', () => {
  for (let top = 0; top < PARTS.top.length; top++) {
    for (let head = 0; head < PARTS.head.length; head++) {
      for (const acc of ['', 'nonla', 'no']) {
        const grid = buildGrid({...randomParts(() => 0.3), top, head}, {frame: 1, acc, effect: 'ngu'});
        assert.ok(grid.slice(0, W).every((c) => !c), `top=${top} head=${head} acc=${acc} touches row 0`);
      }
    }
  }
});
