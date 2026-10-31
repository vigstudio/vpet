const assert = require('node:assert/strict');
const test = require('node:test');
const {MUSIC_KEY, readMusic, writeMusic} = require('./music.cjs');

test('music is opt-in and remembers the choice', () => {
  const values = new Map();
  const storage = {getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, value)};
  assert.equal(readMusic(storage), false);
  writeMusic(storage, true);
  assert.equal(values.get(MUSIC_KEY), 'on');
  assert.equal(readMusic(storage), true);
  writeMusic(storage, false);
  assert.equal(readMusic(storage), false);
});

test('blocked storage never breaks the game', () => {
  const storage = {
    getItem() { throw new Error('denied'); },
    setItem() { throw new Error('denied'); },
  };
  assert.equal(readMusic(storage), false);
  assert.doesNotThrow(() => writeMusic(storage, true));
});
