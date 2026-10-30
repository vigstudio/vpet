import React, {useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import {itemOf} from './store';

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
  const cost = state.gacha_cost || 150;
  const prizes = (state.catalog || []).filter((it) => !['junk', 'bag', 'gem'].includes(it.kind) && (it.rarity || 0) < 4);
  const open = async () => {
    setPhase('loading');
    setLast(null);
    const next = await act(() => api.gacha(), true);
    const won = next?.gacha_result && prizes.find((it) => it.id === next.gacha_result.item);
    if (!won) {
      setPhase('idle');
      return;
    }
    // The first 35 cells are only theatre; the final cell is exactly what the server awarded.
    // Varying the stride changes the visible run without changing odds or the result.
    const stride = 5 + (round % 5) * 2;
    const cells = Array.from({length: 35}, (_, i) => prizes[(i * stride + round * 11) % prizes.length]);
    cells.push(won);
    setReel(cells);
    setRound((n) => n + 1);
    setPhase('spinning');
    setTimeout(() => {
      setLast({item: won, note: next.message});
      setPhase('won');
    }, 3000);
  };
  return (
    <section className='vpet-wheel' aria-label={t('wheel.title')}>
      <header>
        <div><b>{t('wheel.title')}</b><small>{t('wheel.sub', cost)}</small></div>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || phase === 'loading' || phase === 'spinning' || (state.coin || 0) < cost} onClick={open}>
          {(state.coin || 0) < cost ? t('shop.short', cost - (state.coin || 0)) : phase === 'loading' ? t('wheel.locking') : phase === 'spinning' ? t('wheel.spinning') : t('wheel.spin')}
        </button>
      </header>
      <div className='vpet-wheel__window'>
        <i className='vpet-wheel__needle' aria-hidden='true'/>
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

function Shelf({offer, coin, busy, act}) {
  const it = itemOf(offer.item) || {};
  const available = offer.unlimited || offer.left > 0;
  const poor = coin < offer.price;
  return (
    <div className={`vpet-shelf is-${RARITY[it.rarity || 0]}${available ? '' : ' is-out'}`}>
      <ItemIcon id={offer.item} size={34}/>
      <b>{it.name}</b>
      <small>{it.desc}</small>
      <span className='vpet-shelf__price'>{`${offer.price} 🪙`}</span>
      <small className='vpet-muted'>{offer.unlimited ? t('shop.always') : available ? t('shop.left', offer.left, offer.qty) : t('shop.out')}</small>
      <div className='vpet-shelf__actions'>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !available || poor} onClick={() => act(() => api.buy(offer.item, 1))}>
          {available ? (poor ? t('shop.short', offer.price - coin) : t('shop.buy')) : '—'}
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
      <p className='vpet-note'>{t('shop.note', state.coin_left || 0)}</p>
      <Box state={state} busy={busy} act={act}/>
      <section className='vpet-shop__section'>
        <header><div><h4>{t('shop.food')}</h4><small>{t('shop.food.desc')}</small></div><b>{t('shop.items', food.length)}</b></header>
        <div className='vpet-shop__grid is-food'>{food.map((o) => <Shelf key={o.item} offer={o} coin={coin} busy={busy} act={act}/>)}</div>
      </section>
      <section className='vpet-shop__section'>
        <header><div><h4>{t('shop.daily')}</h4><small>{t('shop.daily.desc')}</small></div><b>{t('shop.items', daily.length)}</b></header>
        <div className='vpet-shop__grid'>{daily.map((o) => <Shelf key={o.item} offer={o} coin={coin} busy={busy} act={act}/>)}</div>
      </section>
    </div>
  );
}
