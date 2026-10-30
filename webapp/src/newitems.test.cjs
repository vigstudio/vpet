const assert = require('node:assert/strict');
const test = require('node:test');
const {gainedItems} = require('./newitems.cjs');

test('marks only inventory quantities that increased', () => {
  assert.deepEqual(gainedItems({com: 2, tui: 1}, {com: 3, tui: 0, kiem: 1}).sort(), ['com', 'kiem']);
  assert.deepEqual(gainedItems({}, {}), []);
});
