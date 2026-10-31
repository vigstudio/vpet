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

const {leaderOf, placeOf, roadsOf, stateOf} = require('./alliance.cjs');

const grid = () => Array.from({length: 12}, (_, i) => ({
  id: `land-${String(i + 1).padStart(2, '0')}`,
  x: i % 4,
  y: Math.floor(i / 4),
  adjacent: [],
}));

test('every territory lands inside the board, with room for the castle art', () => {
  grid().forEach((land, i) => {
    const at = placeOf(land, i);
    assert.ok(at.left > 6 && at.left < 94, `left ${at.left}`);
    assert.ok(at.top > 6 && at.top < 94, `top ${at.top}`);
  });
});

test('no two territories are painted on top of each other', () => {
  const seen = grid().map((land, i) => placeOf(land, i)).map((at) => `${at.left.toFixed(1)},${at.top.toFixed(1)}`);
  assert.strictEqual(new Set(seen).size, seen.length);
});

test('the map is the same every render: placement never calls random', () => {
  const land = {x: 2, y: 1};
  assert.deepStrictEqual(placeOf(land, 5), placeOf(land, 5));
});

test('a road is drawn once per adjacent pair, not once per direction', () => {
  const lands = grid();
  lands[0].adjacent = ['land-02'];
  lands[1].adjacent = ['land-01'];
  assert.strictEqual(roadsOf(lands).length, 1);
});

test('an adjacency pointing at a territory that is not on the map is skipped', () => {
  const lands = grid();
  lands[0].adjacent = ['land-99'];
  assert.deepStrictEqual(roadsOf(lands), []);
});

test('the castle says who holds it, from your side', () => {
  assert.strictEqual(stateOf({owner: 'a1'}, 'a1'), 'ours');
  assert.strictEqual(stateOf({owner: 'a2'}, 'a1'), 'theirs');
  assert.strictEqual(stateOf({}, 'a1'), 'neutral');
  assert.strictEqual(stateOf(null, 'a1'), 'neutral');
});

test('the banner names whoever is ahead, and stays empty at a standstill', () => {
  assert.deepStrictEqual(leaderOf({influence: {a1: 40, a2: 90}}), {alliance: 'a2', points: 90});
  assert.strictEqual(leaderOf({influence: {}}), null);
  assert.strictEqual(leaderOf({influence: {a1: 0}}), null);
  assert.strictEqual(leaderOf(undefined), null);
});

test('a tie resolves the same way every render rather than flickering', () => {
  const land = {influence: {b: 50, a: 50}};
  assert.strictEqual(leaderOf(land).alliance, leaderOf(land).alliance);
});
