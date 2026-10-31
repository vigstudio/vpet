import React, {useEffect, useState} from 'react';

import {t, tr} from './i18n';
import music from './music.cjs';
import {audio} from './audio.cjs';

function storage() {
  try { return globalThis.localStorage || {}; } catch (_) { return {}; }
}
function initial() {
  const store = storage();
  let saved = {};
  try { saved = JSON.parse(store.getItem('vpet.audio') || '{}') || {}; } catch (_) { /* blocked storage */ }
  return {
    music: music.readMusic(store), effects: saved.effects === true,
    musicVolume: Number.isFinite(saved.musicVolume) ? Math.max(0, Math.min(1, saved.musicVolume)) : .45,
    effectVolume: Number.isFinite(saved.effectVolume) ? Math.max(0, Math.min(1, saved.effectVolume)) : .6,
  };
}

export default function Music({tab}) {
  const [settings, setSettings] = useState(initial);
  const change = (key, value) => {
    const next = {...settings, [key]: value};
    audio.configure(next);
    audio.unlock();
    setSettings(next);
  };
  useEffect(() => {
    audio.configure(settings);
    music.writeMusic(storage(), settings.music);
    try { storage().setItem('vpet.audio', JSON.stringify(settings)); } catch (_) { /* session still works */ }
  }, [settings]);
  useEffect(() => { audio.scene(tab); }, [tab]);
  useEffect(() => {
    const unlock = () => audio.unlock();
    const visibility = () => audio.visibility(document.hidden);
    document.addEventListener('pointerdown', unlock);
    document.addEventListener('keydown', unlock);
    document.addEventListener('visibilitychange', visibility);
    visibility();
    return () => {
      document.removeEventListener('pointerdown', unlock);
      document.removeEventListener('keydown', unlock);
      document.removeEventListener('visibilitychange', visibility);
      audio.dispose();
    };
  }, []);
  return <fieldset className='vpet-audio'>
    <legend>{tr('Âm thanh', 'Audio', 'サウンド')}</legend>
    <label><input type='checkbox' checked={settings.music} onChange={(e) => change('music', e.target.checked)}/>{t('app.musicOn')}</label>
    <label>{tr('Âm lượng nhạc', 'Music volume', '音楽の音量')}<input type='range' min='0' max='1' step='.05' value={settings.musicVolume} onChange={(e) => change('musicVolume', Number(e.target.value))}/></label>
    <label><input type='checkbox' checked={settings.effects} onChange={(e) => change('effects', e.target.checked)}/>{tr('Tiếng động', 'Sound effects', '効果音')}</label>
    <label>{tr('Âm lượng tiếng động', 'Effects volume', '効果音の音量')}<input type='range' min='0' max='1' step='.05' value={settings.effectVolume} onChange={(e) => change('effectVolume', Number(e.target.value))}/></label>
  </fieldset>;
}
