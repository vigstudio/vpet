const test = require('node:test');
const assert = require('node:assert');
const chat = require('./worldchat.cjs');

test('subscribers see messages and unsubscribe cleanly', () => {
  chat.resetWorldChat();
  const seen = [];
  const off = chat.onWorldChat((rows) => seen.push(rows.map((r) => r.id).join(',')));
  chat.receiveWorldChat({id: 1});
  chat.receiveWorldChat(null);
  off();
  chat.receiveWorldChat({id: 2});
  assert.deepStrictEqual(seen, ['1']);
  assert.strictEqual(chat.worldChatMessages().length, 2, 'the log keeps growing while nobody watches');
});

// A reconnecting stream replays from the cursor, so the same ids arrive twice.
test('a replayed message is not shown twice', () => {
  chat.resetWorldChat();
  chat.receiveWorldChat({id: 7, body: 'hi'});
  chat.receiveWorldChat({id: 7, body: 'hi'});
  assert.strictEqual(chat.worldChatMessages().length, 1);
});

test('the cursor follows the highest id seen', () => {
  chat.resetWorldChat();
  assert.strictEqual(chat.lastWorldChatID(), 0);
  chat.receiveWorldChat({id: 4});
  chat.receiveWorldChat({id: 9});
  chat.receiveWorldChat({id: 6}); // an out-of-order replay must not rewind the cursor
  assert.strictEqual(chat.lastWorldChatID(), 9);
});

test('the backlog seeds both the log and the cursor', () => {
  chat.resetWorldChat();
  chat.seedWorldChat([{id: 11}, {id: 12}]);
  assert.strictEqual(chat.lastWorldChatID(), 12);
  chat.receiveWorldChat({id: 12});
  assert.strictEqual(chat.worldChatMessages().length, 2, 'the backlog overlapping the stream stays deduplicated');
});

// Messages must accumulate while the chat panel is unmounted, which is the normal state on
// a phone; otherwise reopening it shows an empty room until the next HTTP fetch.
test('messages arrive while nothing is listening', () => {
  chat.resetWorldChat();
  chat.receiveWorldChat({id: 1});
  chat.receiveWorldChat({id: 2});
  const seen = [];
  chat.onWorldChat((rows) => seen.push(rows.length));
  chat.receiveWorldChat({id: 3});
  assert.deepStrictEqual(chat.worldChatMessages().map((r) => r.id), [1, 2, 3]);
  assert.deepStrictEqual(seen, [3]);
});

test('the log stays bounded', () => {
  chat.resetWorldChat();
  for (let i = 1; i <= 160; i++) {
    chat.receiveWorldChat({id: i});
  }
  const rows = chat.worldChatMessages();
  assert.strictEqual(rows.length, 100);
  assert.strictEqual(rows[0].id, 61, 'the oldest lines fall off the front');
  assert.strictEqual(chat.lastWorldChatID(), 160);
});
