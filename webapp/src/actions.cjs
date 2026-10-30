// Care actions, mirroring `actions` in internal/game. [id, icon, label key].
// This lived in Buddy.jsx while a pet walked the desktop behind the game; the town screen
// replaced that, so the list now stands on its own rather than dragging a whole screen in.
const ACTIONS = [
  ['vuot', '🤚', 'buddy.stroke'], ['gai', '🐾', 'buddy.scratch'], ['cu', '🤭', 'buddy.tickle'], ['om', '🤗', 'buddy.hug'],
  ['choi', '⚽', 'buddy.ball'], ['tam', '🛁', 'buddy.bathe'], ['ngu', '🌙', 'buddy.sleep'],
  ['bung', '👉', 'buddy.flick'], ['dam', '👊', 'buddy.hit'],
];

module.exports = {ACTIONS};
