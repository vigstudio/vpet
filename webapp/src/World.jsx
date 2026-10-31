import React, {useEffect, useState} from 'react';

import monsters from '../assets/monsters.png';
import tiles from '../assets/world-tiles.png';
import {MONSTERS, TILES} from '../assets/world.js';

import * as api from './api';
import {t} from './i18n';
import {Replay} from './Duel';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import {itemOf, useNow} from './store';
import {REALMS} from './Realm';

// The world map, drawn with Dungeon Crawl Stone Soup tiles (CC0). No walking: pick a region,
// it lights up, the pet searches it, a wild pet jumps out and the fight plays in the arena.
const COLS = 30;
const ROWS = 10;

const BANDS = [
  {id: 'vuon', from: 0, to: 6, ground: ['grass', 'grass2', 'grass3'], props: ['tree', 'tree2'], density: 0.1},
  {id: 'rung', from: 6, to: 12, ground: ['moss', 'grass2', 'grass'], props: ['mangrove', 'tree2', 'tree'], density: 0.25},
  {id: 'hang', from: 12, to: 18, ground: ['cave', 'cave2'], props: ['boulder'], density: 0.12},
  {id: 'nui', from: 18, to: 24, ground: ['snow', 'snow2', 'ice'], props: ['boulder', 'tree2'], density: 0.08},
  {id: 'nuilua', from: 24, to: 30, ground: ['ash', 'lava'], props: ['lava2', 'boulder'], density: 0.14},
];

export const tileStyle = (key) => ({
  backgroundImage: `url(${tiles})`,
  backgroundSize: `${TILES.length * 100}% 100%`,
  backgroundPosition: `${(TILES.indexOf(key) / (TILES.length - 1)) * 100}% 0`,
});

export const monsterStyle = (i) => ({
  backgroundImage: `url(${monsters})`,
  backgroundSize: `${MONSTERS.length * 100}% 100%`,
  backgroundPosition: `${(i / (MONSTERS.length - 1)) * 100}% 0`,
});

function Found({state, wild, onClose, onReplay}) {
  const map = (state.maps || []).find((m) => m.id === wild.map);
  return (
    <div className='vpet-report'>
      <h4>{`${map?.name || ''} — ${wild.won ? t('world.win') : t('world.lose')}`}</h4>
      <p className='vpet-report__note'>{wild.note}</p>
      <div className='vpet-report__loot'>
        {(wild.finds || []).map((f, i) => (
          <span key={`${f.item}-${i}`} className={`vpet-loot is-${RARITY[f.rarity]}`}>
            <ItemIcon id={f.item} size={32}/>
            {itemOf(f.item)?.name || f.item}
            <small>{(state.rarities || [])[f.rarity]}</small>
          </span>
        ))}
      </div>
      <p>{`✨ +${wild.xp} kinh nghiệm`}{wild.coin > 0 && ` · 🪙 +${wild.coin}`}{wild.hurt > 0 && ` · 🩸 mất ${Math.round(wild.hurt)} sức khỏe`}</p>
      <div className='vpet-arena__btns'>
        <button type='button' className='vpet-btn' onClick={onReplay}>{t('hunt.replay')}</button>
        <button type='button' className='vpet-btn vpet-btn--primary' onClick={onClose}>{t('world.continue')}</button>
      </div>
    </div>
  );
}

export default function WorldTab({state, sel, busy, act}) {
  const pet = state.pets[sel];

  const maps = state.maps || [];
  const lv = pet?.lv || 1;
  const [pick, setPick] = useState('vuon');
  const [phase, setPhase] = useState(''); // '', 'search', 'meet'
  const [wild, setWild] = useState(null);
  const [watch, setWatch] = useState(null);
  const region = maps.find((m) => m.id === pick);
  const now = useNow();
  const energy = state.energy?.[sel] || {now: 0, max: 0, wild_cost: 2};
  const refill = energy.next ? Math.max(0, Math.ceil((energy.next - now) / 1000)) : 0;

  // the encounter is a short sequence: searching, then the wild pet jumps out, then the arena
  useEffect(() => {
    if (phase !== 'meet' || !wild) {
      return undefined;
    }
    const t = setTimeout(() => {
      setPhase('');
      setWatch(wild.duel);
    }, 1400);
    return () => clearTimeout(t);
  }, [phase, wild]);

  if (!pet || pet.dead) {
    return <p className='vpet-empty'>{t('world.needPet')}</p>;
  }
  if (watch) {
    return <Replay duel={watch} state={state} onClose={() => setWatch(null)} onComplete={() => setWatch(null)}/>;
  }
  if (wild && !phase) {
    return <Found state={state} wild={wild} onClose={() => setWild(null)} onReplay={() => setWatch(wild.duel)}/>;
  }

  let stop = '';
  if (lv < (region?.min_lv || 1)) {
    stop = t('world.needLevel', region.min_lv);
  } else if (pet.sick) {
    stop = t('hunt.sick');
  } else if (pet.health < 20) {
    stop = t('hunt.weak');
  } else if (pet.hunger < 5) {
    stop = t('hunt.hungry');
  } else if (energy.now < energy.wild_cost) {
    stop = t('world.lowEnergy', energy.now, energy.wild_cost);
  }

  const go = async () => {
    setPhase('search');
    const [next] = await Promise.all([act(() => api.wild(sel, pick), true), new Promise((r) => setTimeout(r, 1300))]);
    if (next?.wild) {
      setWild(next.wild);
      setPhase('meet');
    } else {
      setPhase('');
    }
  };
  const band = BANDS.find((b) => b.id === pick);
  const foe = wild?.duel?.b;
  const span = {left: `${(band.from / COLS) * 100}%`, width: `${((band.to - band.from) / COLS) * 100}%`};

  return (
    <div className='vpet-world'>
      <div className={`vpet-world__board vpet-world__illustrated${phase ? ' is-busy' : ''}`} style={{backgroundImage:`url(${REALMS[pick]})`}}>
        {/* one clickable overlay per region: the picked one glows, the rest dim */}
        {BANDS.map((b, i) => {
          const m = maps.find((x) => x.id === b.id);
          const locked = lv < (m?.min_lv || 1);
          return (
            <button
              key={b.id}
              type='button'
              disabled={Boolean(phase)}
              className={`vpet-region${pick === b.id ? ' is-picked' : ''}${locked ? ' is-locked' : ''}`}
              style={{left: `${(b.from / COLS) * 100}%`, width: `${((b.to - b.from) / COLS) * 100}%`}}
              onClick={() => setPick(b.id)}
              aria-pressed={pick === b.id}
            >
              <span className='vpet-region__foe' style={monsterStyle(i * 5)} aria-hidden='true'/>
              <b>{m?.name}</b>
              <small>{locked ? t('world.lockLevel', m?.min_lv) : `Cấp ${m?.min_lv}+`}</small>
            </button>
          );
        })}
        {phase === 'search' && (
          <div className='vpet-world__fx is-search' style={span}>
            <span>{t('world.searching')}</span>
          </div>
        )}
        {phase === 'meet' && foe && (
          <div className='vpet-world__fx is-meet' style={span}>
            <span className='vpet-world__pop' style={monsterStyle((foe.sprite || 1) - 1)} aria-hidden='true'/>
            <span>{t('world.foe', foe.name)}</span>
          </div>
        )}
      </div>
      <div className='vpet-world__bar'>
        <div>
          <b>{region?.name}</b>
          <small className='vpet-muted'>{region ? ` · ${region.desc} · ${t('world.cost', energy.wild_cost)} · ⚡ ${energy.now}/${energy.max}${refill ? ` · ${t('world.refill', `${Math.floor(refill/60)}:${String(refill%60).padStart(2,'0')}`)}` : ''}` : ''}</small>
        </div>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(phase) || Boolean(stop)} onClick={go}>
          {stop || t('world.adventure')}
        </button>
      </div>
    </div>
  );
}
