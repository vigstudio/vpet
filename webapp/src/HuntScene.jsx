import React, {useEffect, useRef, useState} from 'react';

import PetCanvas from './PetCanvas';
import {RARITY} from './Hunt';
import {beatsOf} from './huntscene.cjs';
import {itemOf} from './store';
import {REALMS} from './Realm';

// Each region is painted with real tiles: a sky band, then rows of that region's ground with
// its trees and rocks standing on it. Same art as the world map (Dungeon Crawl Stone Soup, CC0).
export function Backdrop({id}) {
  return <div className='vpet-scene__bg vpet-scene__illustrated' aria-hidden='true' style={{backgroundImage:`url(${REALMS[id] || REALMS.vuon})`}}/>;
}

// Plays the trip back beat by beat, then hands over to the report.
export default function HuntScene({hunt, map, onDone, onSkip}) {
  const label = (id) => `${itemOf(id)?.emoji || '❔'} ${itemOf(id)?.name || id}`;
  const [beats] = useState(() => beatsOf(hunt, map, label));
  const doneRef = useRef(onDone);
  doneRef.current = onDone;
  const [n, setN] = useState(0);
  const beat = beats[Math.min(n, beats.length - 1)];
  const done = n >= beats.length;

  useEffect(() => {
    if (done) {
      const end = setTimeout(() => doneRef.current(), 400);
      return () => clearTimeout(end);
    }
    const t = setTimeout(() => setN((x) => x + 1), beats[n].ms);
    return () => clearTimeout(t);
  }, [n, done, beats]);

  const pet = hunt.pet || {};
  const cls = ['vpet-scene__hero'];
  if (beat.kind === 'go' || beat.kind === 'home') {
    cls.push('is-walking');
  }
  if (beat.kind === 'find') {
    cls.push('is-digging');
  }
  if (beat.kind === 'hurt' || beat.kind === 'die') {
    cls.push('is-hurt');
  }
  if (beat.kind === 'die') {
    cls.push('is-down');
  }

  return (
    <div className='vpet-scene'>
      <div className={`vpet-scene__stage is-${map?.id || 'vuon'}`}>
        <Backdrop id={map?.id || 'vuon'}/>
        <div className={cls.join(' ')} style={{left: `${beat.x}%`}}>
          <PetCanvas
            pet={{acc: pet.acc, effect: beat.kind === 'hurt' ? 'dau' : '', dead: beat.kind === 'die'}}
            parts={pet.parts}
            scale={3}
          />
          {beat.kind === 'find' && (
            <span key={`f${n}`} className={`vpet-scene__pop is-loot is-${RARITY[beat.rarity] || 'r0'}`} aria-hidden='true'>
              <b>{itemOf(beat.item)?.emoji || '🎁'}</b>
              <small>{itemOf(beat.item)?.name || beat.item}</small>
            </span>
          )}
          {beat.kind === 'look' && <span key={`l${n}`} className='vpet-scene__pop is-ask' aria-hidden='true'>{'?'}</span>}
          {(beat.kind === 'hurt' || beat.kind === 'die') && <span key={`h${n}`} className='vpet-scene__pop is-hurt' aria-hidden='true'>{'!'}</span>}
        </div>
      </div>
      <p className='vpet-scene__log' role='status'>{beat.text}</p>
      <div className='vpet-scene__dots' aria-hidden='true'>
        {beats.map((b, i) => <i key={`${b.kind}-${i}`} className={i <= n ? 'is-on' : ''}/>)}
      </div>
      <button type='button' className='vpet-btn' onClick={onSkip}>{'⏭️ Bỏ qua, xem kết quả'}</button>
    </div>
  );
}
