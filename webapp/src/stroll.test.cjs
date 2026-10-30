const test = require('node:test');
const assert = require('node:assert');
const {newStroll, strollStep, SPEED, FAST} = require('./stroll.cjs');

// A fixed sequence makes the walk reproducible without stubbing the clock.
function seq(values) {
  let i = 0;
  return () => values[i++ % values.length];
}

function run(b, seconds, opts) {
  for (let t = 0; t < seconds; t += 1 / 60) {
    strollStep(b, 1 / 60, opts);
  }
  return b;
}

test('the pet stays inside its lane however long it walks', () => {
  const b = newStroll(50, 20, 80);
  const rnd = seq([0.01, 0.99, 0.5, 0.2, 0.8]);
  for (let t = 0; t < 120; t += 1 / 60) {
    strollStep(b, 1 / 60, {rnd});
    assert.ok(b.x >= 20 && b.x <= 80, `x ${b.x} left the lane`);
  }
});

test('it actually goes somewhere', () => {
  const b = newStroll(50, 10, 90);
  const start = b.x;
  run(b, 20, {rnd: seq([0.9, 0.1])});
  assert.notStrictEqual(b.x, start);
});

test('an excited pet covers more ground in the same time', () => {
  const far = () => 0.999;
  const calm = run(newStroll(10, 10, 90), 2, {rnd: far});
  const hyper = run(newStroll(10, 10, 90), 2, {rnd: far, fast: true});
  assert.ok(hyper.x > calm.x, `fast ${hyper.x} should beat calm ${calm.x}`);
  assert.ok(FAST > SPEED);
});

test('dir reports which way it faces', () => {
  const b = newStroll(80, 10, 90);
  b.wait = 0;
  b.target = 20;
  strollStep(b, 1 / 60, {});
  assert.strictEqual(b.dir, -1, 'walking left');
  b.target = 95;
  strollStep(b, 1 / 60, {});
  assert.strictEqual(b.dir, 1, 'walking right');
});

test('a lane with no room does not make it jitter', () => {
  const b = newStroll(40, 40, 40);
  run(b, 10, {rnd: seq([0.5])});
  assert.strictEqual(b.x, 40);
});

test('it pauses between walks rather than sliding nonstop', () => {
  const b = newStroll(50, 10, 90);
  b.wait = 0;
  b.target = 50.0001; // arrives almost immediately
  strollStep(b, 1 / 60, {rnd: () => 0.5});
  assert.ok(b.wait > 0, 'arriving starts a rest');
});
