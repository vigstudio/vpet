const {test} = require('node:test');
const assert = require('node:assert/strict');
const {createAudio, trackFor, turnCue} = require('./audio.cjs');

function fixture() {
  const contexts = [], timers = new Set();
  const param = () => ({value: 0, setValueAtTime() {}, linearRampToValueAtTime() {}, exponentialRampToValueAtTime() {}, setTargetAtTime(value) { this.value = value; }});
  class Context {
    constructor() { this.state = 'running'; this.currentTime = 0; this.nodes = []; contexts.push(this); }
    createGain() { return {gain: param(), connect() { return this; }, disconnect() {}}; }
    createOscillator() {
      const node = {frequency: param(), connect(gain) { return gain; }, disconnect() {}, start() {}, stop() { this.stopped = true; }};
      this.nodes.push(node);
      return node;
    }
    resume() { this.state = 'running'; return Promise.resolve(); }
    suspend() { this.state = 'suspended'; return Promise.resolve(); }
    close() { this.state = 'closed'; return Promise.resolve(); }
  }
  return {contexts, timers, audio: createAudio({AudioContext: Context, setInterval(fn) { timers.add(fn); return fn; }, clearInterval(fn) { timers.delete(fn); }})};
}

test('scene selection and structured combat cues', () => {
  assert.equal(trackFor('pet'), 'town');
  assert.equal(trackFor('world'), 'adventure');
  const before = [100, 100];
  for (const [skill, cue] of [['lua', 'fire'], ['set', 'spark'], ['', 'hit'], ['khien', 'shield'], ['hoi', 'heal']]) {
    assert.equal(turnCue({skill, by: 0, hp: [100, 90]}, before), cue);
  }
  assert.equal(turnCue({by: 0, hp: before}, before), 'miss');
});

test('opt in, one context, mute, background and teardown', async () => {
  const {audio, contexts, timers} = fixture();
  audio.unlock();
  assert.equal(contexts.length, 0);
  audio.configure({music: true, effects: true, musicVolume: 5, effectVolume: -1});
  audio.unlock();
  audio.unlock();
  await Promise.resolve();
  assert.equal(contexts.length, 1);
  assert.equal(timers.size, 1);
  const ctx = contexts[0];
  assert.ok(ctx.nodes.length > 0 && ctx.nodes.length <= 24);
  audio.visibility(true);
  assert.equal(timers.size, 0);
  assert.ok(ctx.nodes.every((node) => node.stopped));
  const count = ctx.nodes.length;
  audio.cue('win');
  assert.equal(ctx.nodes.length, count);
  audio.visibility(false);
  await Promise.resolve();
  audio.configure({music: false, effects: false});
  assert.equal(timers.size, 0);
  audio.cue('hit');
  assert.equal(ctx.state, 'suspended');
  audio.configure({music: true});
  audio.dispose();
  await Promise.resolve();
  assert.equal(timers.size, 0, 'pending resume must not restart disposed scheduler');
  assert.equal(ctx.state, 'closed');
});

test('effects alone need no music scheduler; unsupported browsers stay safe', async () => {
  const {audio, contexts, timers} = fixture();
  audio.configure({effects: true});
  audio.unlock();
  await Promise.resolve();
  audio.cue('heal');
  assert.equal(contexts[0].nodes.length, 4);
  assert.equal(timers.size, 0);
  audio.dispose();
  const unsupported = createAudio({});
  unsupported.configure({music: true});
  assert.doesNotThrow(() => unsupported.unlock());
});
