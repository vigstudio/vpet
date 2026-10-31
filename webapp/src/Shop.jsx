import React, {useEffect, useRef, useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import {itemOf} from './store';
import {audio} from './audio.cjs';
const {actAt, buildReel, totalMs} = require('./gacha.cjs');

// The daily shop and the blind box. Stock rotates at midnight and is the same for everyone.
export function Coins({state}) {
  return (
    <span className='vpet-coins' title={`Còn kiếm được ${state.coin_left} vàng từ hoạt động hôm nay`}>
      {`🪙 ${(state.coin || 0).toLocaleString('vi-VN')}`}
    </span>
  );
}

function Box({state, busy, act}) {
  const [phase, setPhase] = useState('idle');
  const [last, setLast] = useState(null);
  const [reel, setReel] = useState([]);
  const [round, setRound] = useState(0);
  const [act4, setAct4] = useState('launch');
  const timer = useRef(null);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; clearTimeout(timer.current); };
  }, []);
  const cost = state.gacha_cost || 150;
  const prizes = (state.catalog || []).filter((it) => !['junk', 'bag', 'gem'].includes(it.kind) && (it.rarity || 0) < 4);
  const spinning = phase === 'loading' || phase === 'spinning';

  // The reel scrolls in four acts: it leaps away, blurs, slows, then crawls the last few
  // cells one at a time. The server already chose the prize; only the waiting is built here.
  useEffect(() => {
    if (phase !== 'spinning') {
      return undefined;
    }
    const started = performance.now();
    let lastTick = 0;
    let raf = requestAnimationFrame(function step(now) {
      const at = actAt(now - started);
      setAct4(at);
      const interval = at === 'creep' ? 380 : at === 'slow' ? 210 : 85;
      if (at !== 'done' && now - lastTick >= interval) { audio.cue('tick'); lastTick = now; }
      if (at !== 'done') {
        raf = requestAnimationFrame(step);
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [phase]);

  const open = async () => {
    setPhase('loading');
    setLast(null);
    const next = await act(() => api.gacha(), true);
    if (!mounted.current) return;
    const won = next?.gacha_result && prizes.find((it) => it.id === next.gacha_result.item);
    if (!won) {
      setPhase('idle');
      return;
    }
    const cells = buildReel(prizes, won, round);
    setReel(cells);
    setRound((n) => n + 1);
    setAct4('launch');
    setPhase('spinning');
    timer.current = setTimeout(() => {
      setLast({item: won, note: next.message});
      setPhase('won');
      audio.cue('reward');
    }, window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : totalMs());
  };

  return (
    <section className={`vpet-wheel is-${act4}${spinning ? ' is-live' : ''}`} aria-label={t('wheel.title')}>
      <header>
        <div><b>{t('wheel.title')}</b><small>{t('wheel.sub', cost)}</small></div>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || spinning || (state.coin || 0) < cost} onClick={open}>
          {(state.coin || 0) < cost ? t('shop.short', cost - (state.coin || 0)) : phase === 'loading' ? t('wheel.locking') : spinning ? t('wheel.spinning') : t('wheel.spin')}
        </button>
      </header>
      <div className='vpet-wheel__window'>
        <i className='vpet-wheel__needle' aria-hidden='true'/>
        <i className='vpet-wheel__glow' aria-hidden='true'/>
        {reel.length ? (
          <div key={round} className={`vpet-wheel__reel${phase === 'spinning' ? ' is-spinning' : ''}`} style={{'--winner': reel.length - 1}}>
            {reel.map((it, i) => (
              <div key={`${round}-${i}`} className={`vpet-wheel__cell is-${RARITY[it.rarity || 0]}`} data-item={it.id}>
                <ItemIcon id={it.id} size={34}/><span>{it.name}</span>
              </div>
            ))}
          </div>
        ) : <p>{t('wheel.idle')}</p>}
      </div>
      {/* The state of the spin in words, for anyone who cannot see the reel move. */}
      <p className='vpet-wheel__beat' role='status'>
        {spinning ? t(act4 === 'creep' ? 'wheel.almost' : 'wheel.rolling') : last ? '' : t('wheel.idleHint')}
      </p>
      {last && (
        <p className={`vpet-gacha__won is-${RARITY[last.item.rarity || 0]}`} role='status'>
          <ItemIcon id={last.item.id} size={28}/><span><b>{t('wheel.won', last.item.name)}</b><small>{last.note}</small></span>
        </p>
      )}
      <details className='vpet-wheel__prizes'>
        <summary>{t('wheel.prizes', prizes.length)}</summary>
        <div>{prizes.map((it) => <span key={it.id} className={`is-${RARITY[it.rarity || 0]}`} title={it.name}><ItemIcon id={it.id} size={22}/><small>{it.name}</small></span>)}</div>
      </details>
    </section>
  );
}

// A shelf is a priced good on a counter: the item large, the price on a tag, and the
// reason you cannot buy it said out loud rather than left as a dead button.
function Shelf({offer, coin, busy, act}) {
  const it = itemOf(offer.item) || {};
  const available = offer.unlimited || offer.left > 0;
  const poor = coin < offer.price;
  const left = offer.unlimited ? null : offer.left / Math.max(1, offer.qty);
  return (
    <div className={`vpet-good is-${RARITY[it.rarity || 0]}${available ? '' : ' is-out'}`}>
      <div className='vpet-good__art'><ItemIcon id={offer.item} size={44}/></div>
      <b>{it.name}</b>
      <small className='vpet-good__desc'>{it.desc}</small>
      <span className='vpet-good__tag'>{`${offer.price.toLocaleString('vi-VN')} 🪙`}</span>
      {offer.unlimited
        ? <small className='vpet-good__stock'>{t('shop.always')}</small>
        : <small className='vpet-good__stock'>
            {available ? t('shop.left', offer.left, offer.qty) : t('shop.out')}
            <i aria-hidden='true' style={{width: `${Math.round(Math.max(0, left) * 100)}%`}}/>
          </small>}
      <div className='vpet-good__acts'>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !available || poor} onClick={() => act(() => api.buy(offer.item, 1))}>
          {available ? (poor ? t('shop.short', offer.price - coin) : t('shop.buy')) : t('shop.out')}
        </button>
        {offer.unlimited && <button type='button' className='vpet-btn' disabled={busy || coin < offer.price * 5} onClick={() => act(() => api.buy(offer.item, 5))}>{t('shop.buy5')}</button>}
      </div>
    </div>
  );
}

export default function ShopTab({state, busy, act}) {
  const coin = state.coin || 0;
  const food = (state.shop || []).filter((o) => o.unlimited);
  const daily = (state.shop || []).filter((o) => !o.unlimited);
  return (
    <div className='vpet-shop'>
      {/* The shopkeeper says what is on offer and how much gold the day still owes you. */}
      <header className='vpet-shop__counter'>
        <span className='vpet-shop__keeper' aria-hidden='true'>{'🧙'}</span>
        <div>
          <b>{t('shop.keeper')}</b>
          <small>{t('shop.note', state.coin_left || 0)}</small>
        </div>
        <span className='vpet-shop__purse'>{`🪙 ${coin.toLocaleString('vi-VN')}`}</span>
      </header>
      <Box state={state} busy={busy} act={act}/>
      <section className='vpet-shop__section'>
        <header><div><h4>{t('shop.daily')}</h4><small>{t('shop.daily.desc')}</small></div><b>{t('shop.items', daily.length)}</b></header>
        <div className='vpet-shop__grid'>{daily.map((o) => <Shelf key={o.item} offer={o} coin={coin} busy={busy} act={act}/>)}</div>
      </section>
      <section className='vpet-shop__section'>
        <header><div><h4>{t('shop.food')}</h4><small>{t('shop.food.desc')}</small></div><b>{t('shop.items', food.length)}</b></header>
        <div className='vpet-shop__grid is-food'>{food.map((o) => <Shelf key={o.item} offer={o} coin={coin} busy={busy} act={act}/>)}</div>
      </section>
    </div>
  );
}
