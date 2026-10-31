import React, {useEffect, useRef, useState} from 'react';

import {t} from './i18n';
import music from './music.cjs';

const {readMusic, writeMusic} = music;

// Original 16-second island theme, synthesized in-browser. No audio download, dependency or
// third-party licence; the quiet square/sine mix keeps the loop small and inexpensive.
function makeTrack(context) {
  const seconds = 16;
  const buffer = context.createBuffer(1, context.sampleRate * seconds, context.sampleRate);
  const out = buffer.getChannelData(0);
  const melody = [72,76,79,76,74,77,81,77,72,76,79,83,81,79,76,74];
  const bass = [48,48,53,53,45,45,55,55];
  const hz = (note) => 440 * (2 ** ((note - 69) / 12));
  for (let i = 0; i < out.length; i++) {
    const time = i / context.sampleRate;
    const beat = Math.floor(time * 2);
    const phase = (time * 2) % 1;
    const envelope = Math.min(1, phase * 18) * Math.min(1, (1 - phase) * 5);
    const lead = Math.sin(2 * Math.PI * hz(melody[beat % melody.length]) * time);
    const low = Math.sin(2 * Math.PI * hz(bass[Math.floor(beat / 2) % bass.length]) * time);
    const shimmer = Math.sin(2 * Math.PI * hz(melody[beat % melody.length] + 12) * time);
    out[i] = envelope * (lead * 0.055 + low * 0.035 + shimmer * 0.012);
  }
  return buffer;
}

export default function Music() {
  const [enabled, setEnabled] = useState(() => readMusic(globalThis.localStorage || {}));
  const audio = useRef(null);
  const enabledRef = useRef(enabled);

  const start = async () => {
    if (!enabledRef.current || document.hidden) return;
    if (!audio.current) {
      const AudioContext = globalThis.AudioContext || globalThis.webkitAudioContext;
      if (!AudioContext) return;
      const context = new AudioContext();
      const source = context.createBufferSource();
      const gain = context.createGain();
      source.buffer = makeTrack(context);
      source.loop = true;
      gain.gain.value = 0.72;
      source.connect(gain).connect(context.destination);
      source.start();
      audio.current = {context, source};
    }
    try { await audio.current.context.resume(); } catch (_) { /* browser still wants a gesture */ }
  };

  const stop = async () => {
    if (!audio.current) return;
    try { await audio.current.context.suspend(); } catch (_) { /* already closed */ }
  };

  useEffect(() => {
    enabledRef.current = enabled;
    writeMusic(globalThis.localStorage || {}, enabled);
    if (!enabled) {
      stop();
      return undefined;
    }
    const unlock = () => start();
    document.addEventListener('pointerdown', unlock, {once: true});
    document.addEventListener('keydown', unlock, {once: true});
    return () => {
      document.removeEventListener('pointerdown', unlock);
      document.removeEventListener('keydown', unlock);
    };
  }, [enabled]);

  useEffect(() => {
    const visibility = () => document.hidden ? stop() : start();
    document.addEventListener('visibilitychange', visibility);
    return () => {
      document.removeEventListener('visibilitychange', visibility);
      if (audio.current) audio.current.context.close();
    };
  }, []);

  const toggle = () => {
    const next = !enabled;
    enabledRef.current = next;
    setEnabled(next);
    if (next) start(); else stop();
  };
  const label = enabled ? t('app.musicOff') : t('app.musicOn');
  return <button type='button' className={`vpet-iconbtn vpet-music${enabled ? ' is-on' : ''}`} aria-label={label} title={label} aria-pressed={enabled} onClick={toggle}><span aria-hidden='true'>{enabled ? '♫' : '♪'}</span></button>;
}
