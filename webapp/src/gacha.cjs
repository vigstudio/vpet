// The reel is theatre over a result the server already decided. These helpers decide how
// long each act runs and which cells scroll past, so the tension is testable without a DOM.

// Four acts, in order. The last one is the slow crawl that makes the final cell hurt.
const ACTS = [
  {name: 'launch', ms: 420},
  {name: 'blur', ms: 1500},
  {name: 'slow', ms: 1500},
  {name: 'creep', ms: 1700},
];

const CELLS = 46; // filler cells before the prize; long enough that the blur act never runs out

const totalMs = () => ACTS.reduce((sum, act) => sum + act.ms, 0);

// Builds the strip: filler drawn from the prize pool, the real award last. Round varies the
// stride so two spins in a row do not scroll the identical sequence.
function buildReel(prizes, won, round = 0) {
  if (!prizes?.length || !won) {
    return [];
  }
  const stride = 5 + (round % 5) * 2;
  const cells = Array.from({length: CELLS}, (_, i) => prizes[(i * stride + round * 11) % prizes.length]);
  // A near-miss right before the prize: something rarer than what you won, so the crawl
  // passes a better item and stops one cell late. The tease is visual only.
  const rarer = prizes.filter((it) => (it.rarity || 0) > (won.rarity || 0));
  if (rarer.length) {
    cells[CELLS - 1] = rarer[round % rarer.length];
  }
  cells.push(won);
  return cells;
}

// Which act is running at a given elapsed time. Used for the ticking sound cue and the
// class that slows the blur, so the phases stay in step with the CSS timeline.
function actAt(ms) {
  let seen = 0;
  for (const act of ACTS) {
    seen += act.ms;
    if (ms < seen) {
      return act.name;
    }
  }
  return 'done';
}

module.exports = {ACTS, CELLS, actAt, buildReel, totalMs};
