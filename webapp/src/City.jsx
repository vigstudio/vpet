import {t} from './i18n';
import React, {useEffect, useRef, useState} from 'react';

import Realm from './Realm';
import points from '../assets/realm-points.json';
import PetCanvas from './PetCanvas';
import {newStroll, strollStep} from './stroll.cjs';

// The home screen is a little town: one building per feature, the way the old browser games
// did it. Picking a house opens that tab, so the tab strip no longer has to hold eight labels.
export const HOUSES = [
  {tab: 'pet', x: 20, y: 62},
  {tab: 'hunt', x: 44, y: 40},
  {tab: 'world', x: 70, y: 36},
  {tab: 'shop', x: 33, y: 83},
  {tab: 'bag', x: 57, y: 66},
  {tab: 'market', x: 82, y: 70},
  {tab: 'duel', x: 13, y: 36},
  {tab: 'clinic', x: 62, y: 46},
  {tab: 'grave', x: 90, y: 20},
];

function House({h, i, badge, onGo}) {
  return (
    <button
      type='button'
      className='vpet-house'
      style={{left: `${points.find((p)=>p.id===h.tab)?.x || h.x}%`, top: `${points.find((p)=>p.id===h.tab)?.y || h.y}%`}}
      title={`${t(`house.${h.tab}`)} — ${t(`house.${h.tab}.desc`)}`}
      onClick={() => onGo(h.tab)}
    >
      <span className='vpet-house__art' aria-hidden='true'/>
      <small>{t(`house.${h.tab}`)}</small>
      {badge > 0 && <span className='vpet-badge'>{badge}</span>}
    </button>
  );
}

// The square the pet is free to roam, in percent of the map. It is the open ground below
// the buildings, so a walking pet never ends up behind a house label.
const LANE = {min: 14, max: 86};

// Resident pet. It walks the square on its own; the walk pauses when the tab is hidden or
// the player asked for reduced motion, in which case it simply stands in the plaza.
function Resident({pet}) {
  const [at, setAt] = useState({x: 41, dir: 1});
  const walks = !pet.dead && !pet.sick && pet.effect !== 'ngu';
  const fast = pet.effect === 'cay' || pet.effect === 'hyper';
  const body = useRef(null);

  useEffect(() => {
    const calm = window.matchMedia?.('(prefers-reduced-motion: reduce)');
    if (!walks || calm?.matches) {
      return undefined;
    }
    if (!body.current) {
      body.current = newStroll(41, LANE.min, LANE.max);
    }
    let raf = 0;
    let last = performance.now();
    const frame = (now) => {
      const dt = Math.min((now - last) / 1000, 0.05);
      last = now;
      const b = strollStep(body.current, dt, {fast});
      setAt((prev) => (Math.abs(prev.x - b.x) < 0.05 && prev.dir === b.dir ? prev : {x: b.x, dir: b.dir}));
      raf = requestAnimationFrame(frame);
    };
    raf = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(raf);
  }, [walks, fast]);

  return (
    <div className='vpet-city__pet' style={{left: `${at.x}%`}} aria-hidden='true'>
      <div className={at.dir < 0 ? 'is-flipped' : ''}>
        <PetCanvas pet={pet} parts={pet.parts} scale={2}/>
      </div>
    </div>
  );
}

export default function City({state, sel, onGo}) {
  const pet = state.pets?.[sel];
  const badges = {market: state.in?.length || 0, duel: state.din?.length || 0};
  return (
    <div className='vpet-city'>
      <Realm id='city' className='vpet-city__map'>
        <div className='vpet-realm__heading'><b>{t('city.name')}</b><small>{t('city.motto')}</small></div>
        {HOUSES.map((h, i) => <House key={h.tab} h={h} i={i} badge={badges[h.tab] || 0} onGo={onGo}/>)}
        {pet && !pet.dead && <Resident pet={pet}/>}
      </Realm>
      <p className='vpet-muted'>{t('city.hint')}</p>
    </div>
  );
}
