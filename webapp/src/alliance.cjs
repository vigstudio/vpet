const rankRows = (rows = []) => [...rows].sort((a, b) => a.rank - b.rank);

const influenceFor = (land, alliance) => Number(land?.influence?.[alliance] || 0);

const warCountdown = (ends, now) => {
  const seconds = Math.max(0, Math.ceil((Number(ends || 0) - Number(now || 0)) / 1000));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${String(minutes).padStart(2, '0')}m`;
};


// --- map geometry -------------------------------------------------------------------
// The twelve territories sit on a 4x3 grid on the server. On screen they are a continent:
// each one gets a hand-placed offset so the map reads as terrain rather than a spreadsheet,
// and roads are drawn between adjacent pairs so the front line is visible.

// Jitter per grid cell, in percent of the board. Fixed, not random: the map must look the
// same for everyone, and a moving castle is a map nobody can learn.
const DRIFT = [
  [0, -3], [1.5, 2], [-1.5, -2], [0.5, 3],
  [-2, 1], [2, -2.5], [-1, 2.5], [1.5, -1],
  [0.5, -2.5], [-2, 2], [1, -3], [-0.5, 1.5],
];

// Where a territory is painted, in percent. x/y are the server's grid coordinates.
function placeOf(land, i) {
  const drift = DRIFT[i % DRIFT.length];
  return {
    left: 13 + Number(land?.x || 0) * 24.5 + drift[0],
    top: 20 + Number(land?.y || 0) * 29 + drift[1],
  };
}

// Each adjacency once, so a road is not drawn twice over itself.
function roadsOf(lands = []) {
  const at = new Map(lands.map((land, i) => [land.id, placeOf(land, i)]));
  const seen = new Set();
  const out = [];
  lands.forEach((land, i) => {
    (land.adjacent || []).forEach((id) => {
      const key = [land.id, id].sort().join('>');
      if (seen.has(key) || !at.has(id)) {
        return;
      }
      seen.add(key);
      const from = placeOf(land, i);
      const to = at.get(id);
      out.push({key, x1: from.left, y1: from.top, x2: to.left, y2: to.top});
    });
  });
  return out;
}

// Who is winning a contested territory right now, for the banner over the castle.
function leaderOf(land) {
  const rows = Object.entries(land?.influence || {});
  if (!rows.length) {
    return null;
  }
  const [alliance, points] = rows.sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))[0];
  return points > 0 ? {alliance, points} : null;
}

// How a castle should read at a glance: who holds it and whether it is being fought over.
function stateOf(land, mineID) {
  if (!land) {
    return 'neutral';
  }
  if (land.owner && land.owner === mineID) {
    return 'ours';
  }
  if (land.owner) {
    return 'theirs';
  }
  return 'neutral';
}

module.exports = {DRIFT, influenceFor, leaderOf, placeOf, rankRows, roadsOf, stateOf, warCountdown};
