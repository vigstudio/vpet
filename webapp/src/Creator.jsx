import React, {useState} from 'react';

import * as api from './api';
import PetCanvas from './PetCanvas';
import {t, tr} from './i18n';
import {COLORS, PARTS, PART_LABELS, randomParts} from './sprite.cjs';
import {run} from './store';

const NAMES = ['Bánh Bao', 'Su Hào', 'Mít Ướt', 'Củ Cải', 'Bông Gòn', 'Xoài Non', 'Mập Ú', 'Kẹo Dẻo', 'Bún Chả', 'Tí Nị'];

// Creator hatches a new pet, or with `reskin` (a pet index) spends a transformation card on it.
export default function Creator({onDone, previous, reskin, pet}) {
  const [parts, setParts] = useState(() => (pet ? {...pet.parts} : randomParts()));
  const [name, setName] = useState(() => pet?.name || NAMES[Math.floor(Math.random() * NAMES.length)]);
  const [busy, setBusy] = useState(false);

  const change = (key, delta) => setParts((p) => ({...p, [key]: (p[key] + delta + PARTS[key].length) % PARTS[key].length}));
  const hatch = async () => {
    setBusy(true);
    try {
      await run(() => (reskin === undefined ? api.createPet(name, parts) : api.reskin(reskin, name, parts)));
      onDone?.();
    } catch (e) {
      // message already shown in the bubble
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className='vpet-creator'>
      {reskin !== undefined && <p className='vpet-note'>{t('creator.card')}</p>}
      {previous && <p className='vpet-note'>{`🪦 ${previous} sẽ được an táng trong nghĩa trang.`}</p>}
      <div className='vpet-creator__stage'>
        <PetCanvas parts={parts} scale={6}/>
        <button type='button' className='vpet-btn vpet-btn--ghost' onClick={() => setParts(randomParts())}>{t('creator.random')}</button>
      </div>
      <div className='vpet-creator__parts'>
        {Object.keys(PARTS).map((key) => (
          <div key={key} className='vpet-part'>
            <span className='vpet-part__label'>{t(`creator.part.${key}`)}</span>
            <button type='button' aria-label={t('creator.prev', t(`creator.part.${key}`))} onClick={() => change(key, -1)}>{'◀'}</button>
            <span className='vpet-part__value'>
              {key.startsWith('c') && <i style={{background: COLORS[parts[key]]}}/>}
              {PARTS[key][parts[key]]}
            </span>
            <button type='button' aria-label={t('creator.next', t(`creator.part.${key}`))} onClick={() => change(key, 1)}>{'▶'}</button>
          </div>
        ))}
      </div>
      <label className='vpet-field'>
        {t('creator.name')}
        <input value={name} maxLength={20} onChange={(e) => setName(e.target.value)}/>
      </label>
      <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !name.trim()} onClick={hatch}>{reskin === undefined ? t('creator.hatch') : t('creator.reskin')}</button>
      {onDone && <button type='button' className='vpet-btn' onClick={onDone}>{t('creator.cancel')}</button>}
    </div>
  );
}
