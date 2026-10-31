// Original arrangements. ponytail: small Web Audio ensemble, not sampled orchestral audio.
const THEMES = {
  town: {bpm: 86, roots: [48, 53, 45, 55, 48, 53, 57, 55], lead: [72, 76, 79, 0, 77, 76, 74, 0, 72, 0, 69, 72, 74, 0, 71, 0, 76, 79, 84, 83, 81, 79, 77, 0, 76, 74, 72, 0, 71, 74, 72, 0]},
  adventure: {bpm: 108, roots: [45, 53, 48, 55, 45, 53, 50, 52], lead: [69, 72, 76, 0, 74, 72, 69, 0, 72, 76, 79, 76, 74, 0, 71, 0, 81, 79, 76, 72, 77, 76, 74, 0, 74, 77, 81, 79, 76, 74, 69, 0]},
  battle: {bpm: 138, roots: [45, 45, 53, 52, 45, 48, 50, 52], lead: [69, 0, 76, 75, 76, 72, 69, 0, 77, 76, 72, 69, 71, 74, 76, 0, 81, 79, 76, 0, 77, 76, 74, 72, 74, 0, 77, 76, 71, 68, 69, 0]},
};

function trackFor(tab) {
  return ['hunt', 'world', 'duel'].includes(tab) ? 'adventure' : 'town';
}

function turnCue(turn, before) {
  if (!turn) return '';
  if (turn.skill === 'hoi' || turn.hp?.[turn.by] > before?.[turn.by]) return 'heal';
  if (['khien', 'an'].includes(turn.skill)) return 'shield';
  if (!(turn.hp?.[1 - turn.by] < before?.[1 - turn.by])) return 'miss';
  return ({lua: 'fire', set: 'spark', bom: 'poison', hut: 'poison'})[turn.skill] || 'hit';
}

function createAudio(host = globalThis) {
  let context, musicBus, effectBus, timer, unlocked = false, hidden = false;
  let base = 'town', battle = false, paused = false, track = '', step = 0, next = 0;
  let settings = {music: false, effects: false, musicVolume: .45, effectVolume: .6};
  const voices = new Set();
  const level = (value) => Number.isFinite(Number(value)) ? Math.max(0, Math.min(1, Number(value))) : .5;
  const mode = () => battle ? (paused ? '' : 'battle') : base;
  function stopVoices(bus) {
    for (const voice of [...voices]) {
      if (bus && voice.bus !== bus) continue;
      voices.delete(voice);
      try { voice.osc.stop(); } catch (_) { /* already ended */ }
      voice.osc.disconnect();
      voice.gain.disconnect();
    }
  }
  function tone(note, at, duration, volume, type = 'triangle', bus = musicBus, endNote) {
    if (!context || voices.size >= 24 || !bus) return;
    const osc = context.createOscillator();
    const gain = context.createGain();
    const voice = {osc, gain, bus};
    const hz = (n) => 440 * 2 ** ((n - 69) / 12);
    osc.type = type;
    osc.frequency.setValueAtTime(hz(note), at);
    if (endNote !== undefined) osc.frequency.exponentialRampToValueAtTime(hz(endNote), at + duration);
    gain.gain.setValueAtTime(0, at);
    gain.gain.linearRampToValueAtTime(volume, at + .012);
    gain.gain.exponentialRampToValueAtTime(.0001, at + duration);
    osc.connect(gain).connect(bus);
    voices.add(voice);
    osc.onended = () => { voices.delete(voice); osc.disconnect(); gain.disconnect(); };
    osc.start(at);
    osc.stop(at + duration + .02);
  }
  function schedule() {
    if (!context || context.state !== 'running' || hidden || !settings.music || !mode()) return;
    const theme = THEMES[track];
    const beat = 60 / theme.bpm;
    while (next < context.currentTime + .14) {
      const bar = Math.floor(step / 8) % 8;
      const root = theme.roots[bar];
      const slot = step % 8;
      const phrase = Math.floor(step / 64) % 2;
      const melody = theme.lead[(Math.floor(step / 2) + phrase * 16) % theme.lead.length];
      if (step % 2 === 0 && melody) {
        tone(melody, next, beat * .85, .075, 'triangle');
        tone(melody + 12, next, beat * .5, .014, 'sine');
      }
      if (slot === 0 || (track !== 'town' && slot === 4)) tone(root - 12, next, beat * 1.7, .1, 'sine');
      if (slot === 0) {
        [root + 12, root + (track === 'town' ? 16 : 15), root + 19].forEach((note, i) => tone(note, next + i * .025, beat * 2.8, .026, 'sine'));
      }
      if (track !== 'town') {
        if (slot % 4 === 0) tone(40, next, .16, .14, 'sine', musicBus, 18);
        if (slot % 4 === 2) tone(73, next, .065, .025, 'triangle', musicBus, 43);
        if (track === 'battle' && slot % 2) tone(92, next, .025, .014, 'square', musicBus, 80);
      } else if (slot === 5) tone(root + 26, next, beat * .6, .025, 'sine');
      next += beat / 2;
      step++;
    }
  }
  function sync() {
    if (!context) return;
    const desired = mode();
    if (track !== desired) {
      stopVoices(musicBus);
      track = desired;
      step = 0;
      next = context.currentTime + .03;
    }
    musicBus.gain.setTargetAtTime(settings.music ? settings.musicVolume : 0, context.currentTime, .035);
    effectBus.gain.setTargetAtTime(settings.effects ? settings.effectVolume : 0, context.currentTime, .02);
    if (!settings.music || !desired) stopVoices(musicBus);
    if (!settings.effects) stopVoices(effectBus);
    if (timer) { host.clearInterval(timer); timer = null; }
    if (hidden || (!settings.music && !settings.effects)) {
      stopVoices();
      context.suspend().catch(() => {});
      return;
    }
    if (!unlocked) return;
    next = Math.max(next, context.currentTime + .02);
    const current = context;
    context.resume().then(() => {
      if (context !== current || !unlocked || hidden || !settings.music || !mode()) return;
      if (!timer) timer = host.setInterval(schedule, 60);
      schedule();
    }).catch(() => {});
  }
  return {
    unlock() {
      if (hidden || (!settings.music && !settings.effects)) return;
      unlocked = true;
      if (!context) {
        const AudioContext = host.AudioContext || host.webkitAudioContext;
        if (!AudioContext) return;
        try {
          context = new AudioContext();
          musicBus = context.createGain();
          effectBus = context.createGain();
          musicBus.gain.value = effectBus.gain.value = 0;
          musicBus.connect(context.destination);
          effectBus.connect(context.destination);
        } catch (_) { context = null; return; }
      }
      sync();
    },
    configure(value) {
      settings = {...settings, ...value};
      settings.musicVolume = level(settings.musicVolume);
      settings.effectVolume = level(settings.effectVolume);
      sync();
    },
    scene(tab) { base = trackFor(tab); sync(); },
    battle(active, isPaused = false) { battle = active; paused = isPaused; sync(); },
    visibility(value) { hidden = value; stopVoices(); next = context ? context.currentTime + .03 : 0; sync(); },
    cue(kind) {
      if (!context || context.state !== 'running' || hidden || !settings.effects || !unlocked) return;
      const now = context.currentTime + .005;
      const play = (note, offset, duration, volume, type = 'sine', end) => tone(note, now + offset, duration, volume, type, effectBus, end);
      if (kind === 'tick') play(86, 0, .035, .06, 'triangle');
      else if (kind === 'hit') { play(55, 0, .12, .2, 'triangle', 28); play(80, .025, .055, .05, 'square', 48); }
      else if (kind === 'fire') { play(48, 0, .32, .12, 'sawtooth', 27); play(72, .04, .2, .08, 'triangle', 48); }
      else if (kind === 'spark') [91, 79, 88].forEach((n, i) => play(n, i * .05, .09, .07, 'square', n - 12));
      else if (kind === 'poison') [55, 62, 58].forEach((n, i) => play(n, i * .07, .18, .1, 'sine', n + 7));
      else if (kind === 'shield') [60, 67, 72].forEach((n) => play(n, 0, .4, .08, 'triangle'));
      else if (kind === 'heal') [72, 76, 79, 84].forEach((n, i) => play(n, i * .08, .3, .1));
      else if (kind === 'miss') play(75, 0, .17, .07, 'triangle', 87);
      else if (kind === 'lose' || kind === 'error') [64, 60, 57].slice(0, kind === 'error' ? 2 : 3).forEach((n, i) => play(n, i * .12, .25, .1, 'triangle'));
      else [72, 76, 79, ...(kind === 'win' || kind === 'reward' ? [84, 88] : [])].forEach((n, i) => play(n, i * .085, .3, .08, 'triangle'));
    },
    dispose() {
      if (timer) host.clearInterval(timer);
      timer = null;
      stopVoices();
      if (context) context.close().catch(() => {});
      context = null;
      unlocked = false;
      track = '';
      battle = false;
    },
  };
}

const audio = createAudio();
module.exports = {THEMES, trackFor, turnCue, createAudio, audio};
