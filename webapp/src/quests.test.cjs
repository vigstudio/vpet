const assert = require('node:assert/strict');
const test = require('node:test');

const {ladders, titleGroups, wornTitle} = require('./quests.cjs');

const rung = (track, tier, over) => ({
  id: `a_${track}${tier}`, name: `${track} ${tier}`, desc: '', track, tier, tiers: 3,
  have: 0, need: tier * 10, gold: tier * 100, done: false, daily: false, ...over,
});

test('a ladder shows the rung being worked on, not the finished ones', () => {
  const rows = ladders([
    rung('hunt', 1, {done: true, have: 10}),
    rung('hunt', 2, {done: true, have: 20}),
    rung('hunt', 3, {have: 7}),
  ]);
  assert.equal(rows.length, 1, 'three rungs are one ladder');
  assert.equal(rows[0].quest.tier, 3);
  assert.equal(rows[0].cleared, 2);
  assert.equal(rows[0].tiers, 3);
  assert.equal(rows[0].done, false);
});

test('a finished ladder keeps its top rung instead of disappearing', () => {
  const rows = ladders([
    rung('crush', 1, {done: true}),
    rung('crush', 2, {done: true}),
    rung('crush', 3, {done: true}),
  ]);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].done, true);
  assert.equal(rows[0].cleared, 3);
  assert.equal(rows[0].quest.tier, 3, 'the last rung stays on screen as the record');
});

test('claimable ladders sort above the rest, finished ones sink', () => {
  const rows = ladders([
    rung('a', 1, {done: true}),
    rung('b', 1, {have: 2}),
    rung('c', 1, {have: 10, need: 10}),
  ]);
  assert.deepEqual(rows.map((r) => r.track), ['c', 'b', 'a']);
});

test('dailies never enter the achievement ladders', () => {
  assert.deepEqual(ladders([{id: 'd_feed', daily: true, have: 1, need: 3, done: false}]), []);
});

test('an achievement with no track still renders as its own ladder', () => {
  const rows = ladders([{id: 'a_old', have: 1, need: 5, done: false, daily: false}]);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].track, 'a_old');
  assert.equal(rows[0].tiers, 1);
});

test('rungs out of order are still read lowest tier first', () => {
  const rows = ladders([rung('hunt', 3), rung('hunt', 1), rung('hunt', 2)]);
  assert.equal(rows[0].quest.tier, 1, 'the first unclaimed rung is the lowest one');
});

const title = (track, tier, over) => ({
  id: `t_${track}${tier}`, name: `${track} ${tier}`, desc: '', emoji: '🏅', track, tier,
  need: tier * 10, have: 0, unlocked: false, equipped: false, atk: tier, def: 0, hp: 0, ...over,
});

test('titles group by track and report the best rung earned', () => {
  const groups = titleGroups([
    title('duel', 1, {unlocked: true}),
    title('duel', 2, {unlocked: true}),
    title('duel', 3),
    title('hunt', 1),
  ]);
  assert.equal(groups.length, 2);
  const duel = groups.find((g) => g.track === 'duel');
  assert.equal(duel.unlocked, 2);
  assert.equal(duel.best.tier, 2, 'the best earned rung, not the highest rung');
  assert.equal(duel.rows.length, 3, 'locked rungs stay visible as goals');
  const hunt = groups.find((g) => g.track === 'hunt');
  assert.equal(hunt.best, null, 'a track with nothing earned has no best');
});

test('the worn title is found regardless of track', () => {
  const list = [title('duel', 1, {unlocked: true}), title('hunt', 2, {unlocked: true, equipped: true})];
  assert.equal(wornTitle(list).id, 't_hunt2');
  assert.equal(wornTitle([title('duel', 1)]), null);
  assert.equal(wornTitle(undefined), null, 'an old state with no titles must not throw');
});
