const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

// i18n.js is an ES module the bundler consumes, so it cannot be required here. Reading the
// three locale blocks out of the source is enough to answer the only question that matters:
// does every key Vietnamese defines also exist in English and Japanese? A missing key falls
// back to Vietnamese at runtime, which ships silently and is exactly what this catches.
const source = fs.readFileSync(path.join(__dirname, 'i18n.js'), 'utf8');

// Vietnamese sits inside the STRINGS literal; the other two are assigned onto it afterwards.
function keysOf(locale) {
  const start = locale === 'vi'
    ? source.indexOf(`\n  ${locale}: {`)
    : source.indexOf(`\nSTRINGS.${locale} = {`);
  assert.notEqual(start, -1, `locale block ${locale} not found`);
  let depth = 0;
  let i = source.indexOf('{', start);
  const from = i;
  for (; i < source.length; i++) {
    if (source[i] === '{') depth++;
    else if (source[i] === '}' && --depth === 0) break;
  }
  const block = source.slice(from, i);
  return new Set([...block.matchAll(/'([a-zA-Z][\w.]*)':/g)].map((m) => m[1]));
}

test('every Vietnamese string key exists in English and Japanese', () => {
  const vi = keysOf('vi');
  assert.ok(vi.size > 200, `expected a full key set, found ${vi.size}`);
  for (const locale of ['en', 'ja']) {
    const have = keysOf(locale);
    const missing = [...vi].filter((k) => !have.has(k));
    assert.deepEqual(missing, [], `${locale} is missing ${missing.length} key(s)`);
  }
});

test('the title and quest keys the panel calls are all defined', () => {
  const vi = keysOf('vi');
  const used = [
    'quest.claimed', 'quest.claim', 'quest.notDone', 'quest.note', 'quest.achievements',
    'quest.dailies', 'quest.section', 'quest.ladderHint',
    'title.tab', 'title.none', 'title.noneHint', 'title.rule',
    'title.wear', 'title.worn', 'title.remove',
    ...['lv', 'hunt', 'wild', 'duel', 'star', 'gold', 'arena'].map((x) => `title.track.${x}`),
  ];
  for (const key of used) {
    assert.ok(vi.has(key), `${key} is called by Quests.jsx but never defined`);
  }
});
