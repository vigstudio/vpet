// The script of one hunting trip, built from the server's result so the pet can act it out:
// walk in, sniff around, dig up each find, maybe take a wound, then head home.
// Pure on purpose — the component only plays the beats back.

// x is where the pet stands, 0..100 across the scene.
function beatsOf(hunt, map, label) {
  const name = map?.name || 'vùng đất';
  const beats = [
    {kind: 'go', x: 14, ms: 1700, text: `${map?.emoji || '🗺️'} Lên đường tới ${name}…`},
    {kind: 'look', x: 38, ms: 1500, text: 'Đánh hơi tìm dấu vết…'},
  ];
  const finds = hunt?.finds || [];
  finds.forEach((f, i) => {
    beats.push({
      kind: 'find',
      x: 32 + (((i + 1) * 23) % 46),
      ms: 1700,
      item: f.item,
      rarity: f.rarity,
      text: `Tìm thấy ${label(f.item)}!`,
    });
  });
  if (!finds.length) {
    beats.push({kind: 'look', x: 56, ms: 1500, text: 'Lùng mãi mà chẳng thấy gì…'});
  }
  if (hunt?.hurt > 0) {
    beats.push({kind: 'hurt', x: 64, ms: 1800, text: `🩸 Dính đòn! Mất ${Math.round(hunt.hurt)} sức khỏe.`});
  }
  if (hunt?.note) {
    beats.push({kind: hunt.note.startsWith('💀') ? 'die' : 'note', x: 72, ms: 2000, text: hunt.note});
  }
  const dead = hunt?.note?.startsWith('💀');
  if (!dead) {
    beats.push({kind: 'home', x: 96, ms: 1500, text: `🏠 Về nhà với ${finds.length} món và ✨${hunt?.xp || 0} kinh nghiệm.`});
  }
  return beats;
}

const sceneLength = (beats) => beats.reduce((n, b) => n + b.ms, 0);

module.exports = {beatsOf, sceneLength};
