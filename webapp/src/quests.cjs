// Achievements arrive from the server as a flat list of rungs. Showing all of them turns the
// board into a wall of finished rows, so the panel shows one row per ladder: the rung being
// worked on, or the last one if the ladder is finished. The rungs already claimed are still
// counted, because "3/4" is the progress a player cares about.
//
// This lives in its own module so it can be tested; the panel only renders what it returns.

// ladders groups achievement rows by track, newest-claimable first.
// Each entry: {track, tiers, cleared, quest, done}
//   quest   - the row to render (first unclaimed rung, else the top rung)
//   cleared - how many rungs are already claimed
//   done    - the whole ladder is finished
function ladders(quests) {
  const byTrack = new Map();
  for (const q of quests) {
    if (q.daily) continue;
    // A row with no track is an achievement from before ladders existed; treat it as a
    // one-rung ladder of its own so it still renders instead of vanishing.
    const key = q.track || q.id;
    if (!byTrack.has(key)) byTrack.set(key, []);
    byTrack.get(key).push(q);
  }
  const out = [];
  for (const [track, rows] of byTrack) {
    const sorted = [...rows].sort((a, b) => (a.tier || 0) - (b.tier || 0) || a.need - b.need);
    const cleared = sorted.filter((q) => q.done).length;
    const next = sorted.find((q) => !q.done);
    out.push({
      track,
      tiers: sorted[0]?.tiers || sorted.length,
      cleared,
      quest: next || sorted[sorted.length - 1],
      done: !next,
    });
  }
  // A claimable reward is the only thing a player must act on, so it sorts to the top;
  // finished ladders sink to the bottom.
  const ready = (l) => !l.done && l.quest.have >= l.quest.need;
  return out.sort((a, b) => Number(ready(b)) - Number(ready(a)) || Number(a.done) - Number(b.done));
}

// titleGroups turns the flat title ladder into one group per track, in the server's order,
// so the panel can render a column per track rather than 35 loose cards.
function titleGroups(titles) {
  const byTrack = new Map();
  for (const t of titles || []) {
    if (!byTrack.has(t.track)) byTrack.set(t.track, []);
    byTrack.get(t.track).push(t);
  }
  return [...byTrack].map(([track, rows]) => {
    const sorted = [...rows].sort((a, b) => a.tier - b.tier);
    return {
      track,
      emoji: sorted[0]?.emoji || '🏅',
      rows: sorted,
      unlocked: sorted.filter((t) => t.unlocked).length,
      // the best rung already earned is what the player would actually wear
      best: [...sorted].reverse().find((t) => t.unlocked) || null,
    };
  });
}

// wornTitle is the title currently equipped, or null.
function wornTitle(titles) {
  return (titles || []).find((t) => t.equipped) || null;
}

module.exports = {ladders, titleGroups, wornTitle};
