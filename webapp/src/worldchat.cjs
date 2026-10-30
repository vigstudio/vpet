// World chat lives here rather than in the component so messages keep arriving while the
// chat is closed (the phone layout unmounts it), and so a reconnecting stream knows which
// line the player already has.

const listeners = new Set();
let messages = [];
let lastID = 0;
const LIMIT = 100;

// receiveWorldChat records a message and tells the screen. Ids repeat whenever the stream
// replays after a reconnect, so duplicates are dropped here rather than in the component.
function receiveWorldChat(message) {
  if (!message || !message.id || messages.some((row) => row.id === message.id)) {
    return;
  }
  messages = messages.concat(message).slice(-LIMIT);
  lastID = Math.max(lastID, message.id);
  listeners.forEach((listener) => listener(messages));
}

// seedWorldChat replaces the log with the backlog fetched over HTTP.
function seedWorldChat(rows) {
  messages = (rows || []).slice(-LIMIT);
  lastID = messages.reduce((top, row) => Math.max(top, row.id || 0), lastID);
  listeners.forEach((listener) => listener(messages));
}

function worldChatMessages() {
  return messages;
}

// lastWorldChatID is the resume point handed to the stream so a reconnect replays what was
// said while the connection was down.
function lastWorldChatID() {
  return lastID;
}

function onWorldChat(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function resetWorldChat() {
  messages = [];
  lastID = 0;
  listeners.clear();
}

module.exports = {receiveWorldChat, seedWorldChat, worldChatMessages, lastWorldChatID, onWorldChat, resetWorldChat};
