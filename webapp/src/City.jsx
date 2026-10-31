import {t} from './i18n';
import React, {useEffect, useRef, useState} from 'react';

import cityScene from '../assets/fantasy-city.webp';
import home from '../assets/fantasy/building-home.png';
import hunt from '../assets/fantasy/building-hunt.png';
import portal from '../assets/fantasy/building-portal.png';
import shop from '../assets/fantasy/building-shop.png';
import warehouse from '../assets/fantasy/building-warehouse.png';
import market from '../assets/fantasy/building-market.png';
import arena from '../assets/fantasy/building-arena.png';
import clinic from '../assets/fantasy/building-clinic.png';
import grave from '../assets/fantasy/building-grave.png';
import points from '../assets/realm-points.json';
import PetCanvas from './PetCanvas';
import {newStroll, strollStep} from './stroll.cjs';

const BUILDINGS = {pet: home, hunt, world: portal, shop, bag: warehouse, market, duel: arena, clinic, grave};

// Each town building opens its existing feature tab.
export const HOUSES = points.map(({id, x, y}) => ({tab: id, x, y}));

function House({h, badge, onGo}) {
  return (
    <button
      type='button'
      className='vpet-house'
      style={{left: `${h.x}%`, top: `${h.y}%`}}
      title={`${t(`house.${h.tab}`)} — ${t(`house.${h.tab}.desc`)}`}
      onClick={() => onGo(h.tab)}
    >
      <img className='vpet-house__art' src={BUILDINGS[h.tab]} alt='' aria-hidden='true'/>
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
      <div className='vpet-realm vpet-city__map vpet-city__fantasy' style={{backgroundImage: `url(${cityScene})`}}>
        <div className='vpet-realm__heading'><b>{t('city.name')}</b><small>{t('city.motto')}</small></div>
        {HOUSES.map((h) => <House key={h.tab} h={h} badge={badges[h.tab] || 0} onGo={onGo}/>)}
        {pet && !pet.dead && <Resident pet={pet}/>}
      </div>
      <nav className='vpet-city__doors' aria-label={t('tab.city')}>
        {HOUSES.map((h) => <button key={h.tab} type='button' onClick={() => onGo(h.tab)}>{t(`house.${h.tab}`)}{badges[h.tab] > 0 ? ` (${badges[h.tab]})` : ''}</button>)}
      </nav>
      <p className='vpet-muted'>{t('city.hint')}</p>
    </div>
  );
}
