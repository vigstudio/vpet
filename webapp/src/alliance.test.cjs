const test = require('node:test');
const assert = require('node:assert/strict');
const {influenceFor, rankRows, warCountdown} = require('./alliance.cjs');

test('sorts ranks without mutating input', () => {
  const rows = [{rank: 2}, {rank: 1}];
  assert.deepEqual(rankRows(rows), [{rank: 1}, {rank: 2}]);
  assert.equal(rows[0].rank, 2);
});

test('reads influence safely', () => {
  assert.equal(influenceFor({influence: {a: 125}}, 'a'), 125);
  assert.equal(influenceFor({}, 'a'), 0);
});

test('formats countdown and floors expired time', () => {
  assert.equal(warCountdown(7_300_000, 0), '2h 01m');
  assert.equal(warCountdown(0, 1), '0h 00m');
});
