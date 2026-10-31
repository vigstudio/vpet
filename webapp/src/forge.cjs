// The forge used to be grouped by mechanism: fusing on one screen, enhancing stacked above
// it, socketing on a third. A player does not think "I want to socket", they think "make this
// sword better" - so the bench is grouped by the item instead, and every mechanic that can
// touch the selected piece is listed against it.
//
// This module answers two questions and nothing else: what can be put on the bench, and what
// can be done to the piece that is on it. The panel renders the answer.

// benchItems lists every piece of gear the player owns, worn first, then enhanced stacks,
// then plain bag copies. One entry per {item, level} so a +0 and a +3 sword are separate
// rows - they really are separate things to the server.
function benchItems({worn = [], inv = {}, enhanced = [], gearStats = {}, catalog = []}) {
  const named = (id) => catalog.find((c) => c.id === id);
  const rows = [];
  for (const view of worn) {
    const item = named(view.item);
    if (item) rows.push({key: `worn:${view.slot}`, item, level: view.level || 0, count: 1, slot: view.slot, worn: true});
  }
  for (const stack of enhanced) {
    const item = named(stack.item);
    if (item && stack.count > 0) rows.push({key: `bag:${stack.item}:${stack.level}`, item, level: stack.level, count: stack.count, slot: gearStats[stack.item]?.slot, worn: false});
  }
  for (const [id, count] of Object.entries(inv)) {
    const item = named(id);
    if (!item || count <= 0 || !gearStats[id]) continue;
    rows.push({key: `bag:${id}:0`, item, level: 0, count, slot: gearStats[id].slot, worn: false});
  }
  return rows.sort((a, b) =>
    Number(b.worn) - Number(a.worn) ||
    (b.item.rarity || 0) - (a.item.rarity || 0) ||
    b.level - a.level ||
    a.item.name.localeCompare(b.item.name));
}

// benchActions describes the three things that can be done to the selected piece. Each action
// always reports where it is now, what it would become and what it costs; an action that
// cannot run says why instead of vanishing, because a missing button teaches nothing.
function benchActions(row, {enhance = [], sockets = [], recipes = [], coin = 0, shards = 0, catalog = []}) {
  if (!row) return [];
  const named = (id) => catalog.find((c) => c.id === id);
  const out = [];

  // 1. enhance - worn gear only, because the level lives on the slot
  const view = row.worn ? enhance.find((e) => e.slot === row.slot) : null;
  if (!row.worn) {
    out.push({kind: 'enhance', blocked: 'equip', level: row.level});
  } else if (view && view.level >= view.max) {
    out.push({kind: 'enhance', blocked: 'max', level: view.level, max: view.max});
  } else if (view) {
    out.push({
      kind: 'enhance', slot: row.slot, level: view.level, max: view.max,
      current: view.current, next: view.next,
      coin: view.coin, shards: view.shards,
      blocked: view.affordable ? '' : 'cost',
      shortCoin: Math.max(0, view.coin - coin),
      shortShards: Math.max(0, view.shards - shards),
    });
  }

  // 2. sockets - also worn only, since a gem seats into a worn piece
  if (!row.worn) {
    out.push({kind: 'socket', blocked: 'equip'});
  } else {
    const seat = sockets.find((s) => s.slot === row.slot);
    out.push(seat
      ? {kind: 'socket', slot: row.slot, gems: seat.gems, free: seat.free, blocked: seat.free > 0 ? '' : 'full'}
      : {kind: 'socket', blocked: 'equip'});
  }

  // 3. fuse - bag copies at +0 only; an enhanced piece has to be restored first or its level
  // would be destroyed by a payload that only carries the item id
  const recipe = recipes.find((r) => r.from === row.item.id);
  if (row.level > 0) {
    out.push({kind: 'fuse', blocked: 'enhanced', level: row.level});
  } else if (!recipe) {
    out.push({kind: 'fuse', blocked: 'top'});
  } else {
    out.push({
      kind: 'fuse', recipe, into: named(recipe.to),
      have: recipe.have, need: recipe.need, cost: recipe.cost,
      blocked: recipe.ready ? '' : (recipe.have < recipe.need ? 'parts' : recipe.material && recipe.material_have < recipe.material_need ? 'material' : 'cost'),
      shortParts: Math.max(0, recipe.need - recipe.have),
      shortCoin: Math.max(0, recipe.cost - coin),
    });
  }

  // 4. restore - only offered when there is a level to give back
  if (row.level > 0 && !row.worn) {
    out.push({kind: 'restore', item: row.item.id, level: row.level});
  }
  return out;
}

module.exports = {benchItems, benchActions};
