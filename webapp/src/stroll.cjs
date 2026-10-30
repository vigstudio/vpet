// Strolling across the town square. Coordinates are percentages of the map, so the walk
// works at any map size without the caller measuring pixels: the same state drives a
// 320 px phone and a wide desktop.
//
// There is no gravity here, only "pick a spot, walk there, stand a while".

const SPEED = 6.5; // percent of the map width per second
const FAST = 15; // spicy or hyper pets trot
const PAUSE_MIN = 1.2;
const PAUSE_MAX = 4.5;
const STEP_MIN = 8; // never pick a target so close the pet twitches in place

// newStroll seeds a walker standing still at x, free to roam between min and max.
function newStroll(x, min, max) {
  return {x, min, max, target: x, dir: 1, wait: PAUSE_MIN};
}

// pickTarget chooses somewhere else to be, at least STEP_MIN away when the lane allows it.
function pickTarget(b, rnd) {
  const span = b.max - b.min;
  if (span <= 0) {
    return b.min;
  }
  for (let i = 0; i < 4; i++) {
    const t = b.min + (rnd() * span);
    if (Math.abs(t - b.x) >= Math.min(STEP_MIN, span)) {
      return t;
    }
  }
  return b.x < (b.min + b.max) / 2 ? b.max : b.min;
}

// strollStep advances the walker by dt seconds and returns it. `fast` doubles the pace for
// an excited pet; `rnd` is injectable so the test can make the walk deterministic.
function strollStep(b, dt, {fast = false, rnd = Math.random} = {}) {
  b.x = Math.min(Math.max(b.x, b.min), b.max);
  if (b.wait > 0) {
    b.wait -= dt;
    if (b.wait <= 0) {
      b.wait = 0;
      b.target = pickTarget(b, rnd);
    }
    return b;
  }
  const gap = b.target - b.x;
  const move = (fast ? FAST : SPEED) * dt;
  if (Math.abs(gap) <= move) {
    b.x = b.target;
    b.wait = PAUSE_MIN + (rnd() * (PAUSE_MAX - PAUSE_MIN));
    return b;
  }
  b.dir = gap > 0 ? 1 : -1;
  b.x += b.dir * move;
  return b;
}

module.exports = {newStroll, strollStep, SPEED, FAST, PAUSE_MIN, PAUSE_MAX};
