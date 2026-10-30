import React, {useEffect, useRef, useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import {itemOf, refresh} from './store';

// The player market: two halves of one screen. The left is the board everyone can buy from,
// the right is the stall where you price your own goods. Both are visible at once so the
// decision "is my price fair" has the answer sitting next to it.

const unit = (row) => Math.round(row.price / Math.max(row.qty, 1));

const KINDS = ['food', 'med', 'energy', 'acc', 'card', 'gear', 'bag', 'stone', 'gem', 'revive'];

// A listing used to be a name and a price, which says nothing about what the item does.
// Every row now carries the same facts the bag shows: what it is, what it does, what it
// adds when worn, and how the asking price compares with the shop.
// Picking by name alone made it impossible to tell two similar items apart. The picker
// shows the same icon, rarity border and count the bag uses, so the choice is visual.
function ItemPicker({items, state, value, busy, label, onPick}) {
  return (
    <div className='vpet-market__picker' role='group' aria-label={label}>
      {items.map((it) => (
        <button key={it.id} type='button' aria-pressed={value === it.id} disabled={busy}
          className={`vpet-slot is-${RARITY[it.rarity || 0]}${value === it.id ? ' is-picked' : ''}`}
          title={`${it.name} — ${it.desc || ''}`} aria-label={`${it.name}${state.inv[it.id] ? `, có ${state.inv[it.id]}` : ''}`}
          onClick={() => onPick(it.id)}>
          <ItemIcon id={it.id} size={30}/>
          <span className='vpet-slot__name'>{it.name}</span>
          {state.inv[it.id] > 0 && <span className='vpet-slot__n'>{`×${state.inv[it.id]}`}</span>}
        </button>
      ))}
      {!items.length && <p className='vpet-inv__empty'>{t('market.bagEmpty')}</p>}
    </div>
  );
}

function RequestPicker({items, state, value, busy, onPick}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const search = useRef(null);
  const chosen = items.find((it) => it.id === value) || items[0];
  const q = query.trim().toLocaleLowerCase();
  const matches = (q ? items.filter((it) => it.name.toLocaleLowerCase().includes(q)) : items).slice(0, 60);

  useEffect(() => {
    if (open) search.current?.focus();
  }, [open]);

  const pick = (id) => {
    onPick(id);
    setOpen(false);
    setQuery('');
  };

  return (
    <div className='vpet-request-picker'>
      <button type='button' className='vpet-request-picker__selected' disabled={busy}
        aria-expanded={open} aria-controls='vpet-request-picker-list'
        onClick={() => setOpen(!open)}>
        {chosen && <ItemIcon id={chosen.id} size={36}/>}
        <span>
          <b>{chosen?.name || t('market.blocked.nothing')}</b>
          <small>{chosen?.desc}</small>
        </span>
        <i aria-hidden='true'>{open ? '▲' : '▼'}</i>
      </button>
      {open && (
        <div className='vpet-request-picker__panel'>
          <label className='vpet-inv__search-label'>{t('market.findItem')}
            <input ref={search} className='vpet-inv__search' type='search' value={query}
              placeholder={t('market.findItemHint')} onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Escape') setOpen(false); }}/>
          </label>
          <div id='vpet-request-picker-list' className='vpet-request-picker__list' role='listbox' aria-label={t('market.itemWanted')}>
            {matches.map((it) => (
              <button key={it.id} type='button' role='option' aria-selected={chosen?.id === it.id}
                className={`vpet-request-picker__option is-${RARITY[it.rarity || 0]}`} onClick={() => pick(it.id)}>
                <ItemIcon id={it.id} size={30}/>
                <span><b>{it.name}</b><small>{`${state.rarities?.[it.rarity || 0] || ''}${it.desc ? ` · ${it.desc}` : ''}`}</small></span>
                {chosen?.id === it.id && <i aria-hidden='true'>✓</i>}
              </button>
            ))}
            {!matches.length && <p className='vpet-inv__empty'>{t('market.noMatch')}</p>}
          </div>
        </div>
      )}
    </div>
  );
}

function Detail({state, row}) {
  const it = itemOf(row.item) || {};
  const gear = (state.gear_stats || {})[row.item];
  const shop = (state.prices || {})[row.item] || 0;
  const per = unit(row);
  const stats = gear && [
    gear.atk ? `⚔️ +${gear.atk}` : '',
    gear.def ? `🛡️ +${gear.def}` : '',
    gear.hp ? `❤️ +${gear.hp}` : '',
  ].filter(Boolean).join(' · ');
  return (
    <div className='vpet-market__detail'>
      {it.desc && <p>{it.desc}</p>}
      <span className='vpet-market__tags'>
        {KINDS.includes(it.kind) && <i>{t(`kind.${it.kind}`)}</i>}
        {gear?.slot && <i>{(state.gear_slots || []).find((g) => g.id === gear.slot)?.name || gear.slot}</i>}
        {stats && <i className='is-stat'>{stats}</i>}
        {shop > 0 && <i className={per <= shop ? 'is-deal' : ''}>{t('market.shopPrice', shop)}</i>}
      </span>
    </div>
  );
}

function Row({row, state, busy, onBuy, onCancel}) {
  const it = itemOf(row.item) || {};
  const afford = (state.coin || 0) >= row.price;
  return (
    <li className={`vpet-market__row is-${RARITY[it.rarity || 0]}`}>
      <ItemIcon id={row.item} size={36}/>
      <div className='vpet-market__what'>
        <b>{`${it.name || row.item}${row.qty > 1 ? ` ×${row.qty}` : ''}`}</b>
        <small>{`${state.rarities?.[it.rarity || 0] || ''} · ${row.seller_name}`}</small>
        <Detail state={state} row={row}/>
      </div>
      <div className='vpet-market__price'>
        <b>{`${row.price} 🪙`}</b>
        <small>{t('market.perItem', unit(row))}</small>
      </div>
      {row.mine ? (
        <button type='button' className='vpet-btn' disabled={busy} onClick={() => onCancel(row.id)}>{t('market.pull')}</button>
      ) : (
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !afford} onClick={() => onBuy(row.id)}>{afford ? t('market.buy') : t('market.poor')}</button>
      )}
    </li>
  );
}

function WantRow({row, state, busy, onFulfill, onCancel}) {
  const it = itemOf(row.item) || {};
  const have = state.inv[row.item] || 0;
  const earns = row.price - Math.max(1, Math.floor(row.price * (state.market_rules?.[1] || 10) / 100));
  return (
    <li className={`vpet-market__row vpet-market__want is-${RARITY[it.rarity || 0]}`}>
      <ItemIcon id={row.item} size={36}/>
      <div className='vpet-market__what'>
        <b>{`${it.name || row.item}${row.qty > 1 ? ` ×${row.qty}` : ''}`}</b>
        <small>{`${t('market.needs')} · ${row.buyer_name} · ${t('market.youHave', have)}`}</small>
        <Detail state={state} row={row}/>
      </div>
      <div className='vpet-market__price'>
        <b>{`${row.price} 🪙`}</b>
        <small>{row.mine ? t('market.held') : t('market.earns', earns)}</small>
      </div>
      {row.mine ? (
        <button type='button' className='vpet-btn' disabled={busy} onClick={() => onCancel(row.id)}>{t('market.cancelWant')}</button>
      ) : (
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || have < row.qty} onClick={() => onFulfill(row.id)}>
          {have < row.qty ? t('market.short', row.qty - have) : t('market.sellNow')}
        </button>
      )}
    </li>
  );
}

export default function MarketTab({state, busy, act, children, sellItem, onSellItemUsed}) {
  const [view, setView] = useState('sell');
  const [query, setQuery] = useState('');
  const [item, setItem] = useState('');
  const [qty, setQty] = useState(1);
  const [price, setPrice] = useState('');
  const [wantItem, setWantItem] = useState('');
  const [wantQty, setWantQty] = useState(1);
  const [wantPrice, setWantPrice] = useState('');
  const [note, setNote] = useState('');
  const [onlyAfford, setOnlyAfford] = useState(false);
  const [confirm, setConfirm] = useState(null);

  // arriving from the bag: preselect that item and open the sell form on it
  useEffect(() => {
    if (!sellItem) return;
    setView('sell');
    setItem(sellItem);
    onSellItemUsed?.();
  }, [sellItem]);

  const [fee, tax, ceiling] = state.market_rules || [5, 10, 5];
  const rows = (state.market || []).filter((r) => {
    const it = itemOf(r.item) || {};
    const text = `${it.name || ''} ${r.seller_name}`.toLocaleLowerCase();
    return text.includes(query.toLocaleLowerCase()) && (!onlyAfford || r.mine || (state.coin || 0) >= r.price);
  });
  const mine = (state.market || []).filter((r) => r.mine);
  const orders = (state.buy_orders || []).filter((r) => {
    const it = itemOf(r.item) || {};
    return `${it.name || ''} ${r.buyer_name}`.toLocaleLowerCase().includes(query.toLocaleLowerCase());
  });
  const myOrders = (state.buy_orders || []).filter((r) => r.mine);
  const requestable = state.catalog.filter((it) => it.kind !== 'junk');
  const wanted = requestable.find((it) => it.id === wantItem) || requestable[0];
  const requestedQty = Math.min(Math.max(Number(wantQty) || 1, 1), 99);
  const bid = Number(wantPrice) || 0;
  const maxBid = (state.prices?.[wanted?.id] || 0) * requestedQty * ceiling;
  const wantBlocked = !wanted ? t('market.blocked.nothing')
    : !Number.isInteger(Number(wantQty)) || Number(wantQty) < 1 || Number(wantQty) > 99 ? t('market.blocked.qtyWant')
      : !Number.isInteger(bid) || bid < 1 ? t('market.blocked.price')
        : bid > maxBid ? t('market.blocked.maxWant', maxBid)
          : myOrders.length >= 8 ? t('market.blocked.wantSlots')
            : (state.coin || 0) < bid + fee ? t('market.blocked.wantCoin', bid + fee, fee) : '';

  // only sellable goods, and only what is actually in the bag right now
  const stock = state.catalog.filter((it) => state.inv[it.id] > 0 && it.kind !== 'junk');
  const chosen = stock.find((it) => it.id === item) || stock[0];
  const have = chosen ? state.inv[chosen.id] : 0;
  const lot = Math.min(Math.max(Number(qty) || 1, 1), have || 1);
  const asking = Number(price) || 0;
  const maxPrice = (state.prices?.[chosen?.id] || 0) * lot * ceiling;
  const earns = asking ? asking - Math.max(1, Math.floor(asking * tax / 100)) : 0;
  const blocked = !chosen ? t('market.blocked.noGoods')
    : (state.coin || 0) < fee ? t('market.blocked.fee', fee, state.coin || 0)
      : !Number.isInteger(Number(qty)) || Number(qty) < 1 || Number(qty) > Math.min(have, 99) ? t('market.blocked.qty')
      : !Number.isInteger(asking) || asking < 1 ? t('market.blocked.price')
      : asking > maxPrice ? t('market.blocked.max', maxPrice)
        : mine.length >= 8 ? t('market.blocked.slots') : '';

  const buy = async (id) => {
    const next = await act(() => api.buyListing(id), true);
    setConfirm(null);
    if (next) setNote(next.message);
    else await refresh();
  };
  const cancel = async (id) => {
    const next = await act(() => api.cancelListing(id), true);
    if (next) setNote(next.message);
  };
  const sell = async () => {
    if (busy || blocked) return;
    const next = await act(() => api.sellItem(chosen.id, lot, asking), true);
    if (next) {
      setNote(next.message);
      setConfirm(null);
      setPrice('');
      setQty(1);
    }
  };
  const postWant = async () => {
    if (busy || wantBlocked) return;
    const next = await act(() => api.postBuyOrder(wanted.id, requestedQty, bid), true);
    if (next) {
      setNote(next.message);
      setConfirm(null);
      setWantPrice('');
      setWantQty(1);
    }
  };
  const fulfill = async (id) => {
    const next = await act(() => api.fulfillBuyOrder(id), true);
    if (next) setNote(next.message);
    else await refresh();
  };
  const cancelWant = async (id) => {
    const next = await act(() => api.cancelBuyOrder(id), true);
    if (next) setNote(next.message);
  };

  return (
    <div className='vpet-market'>
      <p className='vpet-note'>{t('market.note', fee, tax)}</p>
      {note && <div className='vpet-inv__receipt' role='status'>{note}</div>}
      <nav className='vpet-market__modes' role='tablist' aria-label={t('market.modes')}>
        <button type='button' role='tab' aria-selected={view === 'sell'} onClick={() => { setView('sell'); setConfirm(null); setQuery(''); }}>{t('market.selling', (state.market || []).length)}</button>
        <button type='button' role='tab' aria-selected={view === 'want'} onClick={() => { setView('want'); setConfirm(null); setQuery(''); }}>{t('market.wanted', (state.buy_orders || []).length)}</button>
      </nav>
      {view === 'sell' && confirm?.kind === 'buy' && (() => {
        const row = (state.market || []).find((r) => r.id === confirm.id);
        const it = row && itemOf(row.item);
        return <section className='vpet-market__confirm' aria-label={t('market.confirmBuy')}>
          <b>{it?.name || t('market.gone')}</b><p>{it?.desc}</p>
          {row && <p>{t('market.buyLine', row.qty, row.seller_name, row.price, (state.coin || 0) - row.price)}</p>}
          <button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm(null)}>{t('app.cancel')}</button>
          <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !row || (state.coin || 0) < row.price} onClick={() => buy(row.id)}>{t('market.confirmBuy')}</button>
        </section>;
      })()}
      {view === 'sell' && (
        <div className='vpet-market__body'>
        <section className='vpet-market__board' aria-label={t('market.board')}>
          <div className='vpet-market__tools'>
            <button type='button' className='vpet-btn' disabled={busy} onClick={() => { setConfirm(null); refresh(); }}>{t('app.refresh')}</button>
            <label className='vpet-inv__search-label'>
              {t('market.search')}
              <input className='vpet-inv__search' type='search' placeholder={t('market.searchHint')} value={query} onChange={(e) => setQuery(e.target.value)}/>
            </label>
            <label className='vpet-market__filter'>
              <input type='checkbox' checked={onlyAfford} onChange={(e) => setOnlyAfford(e.target.checked)}/>
              {t('market.affordOnly')}
            </label>
          </div>
          {rows.length ? (
            <ul className='vpet-market__list'>
              {rows.map((r) => <Row key={r.id} row={r} state={state} busy={busy} onBuy={(id) => setConfirm({kind: 'buy', id})} onCancel={cancel}/>)}
            </ul>
          ) : <p className='vpet-empty'>{query || onlyAfford ? t('market.noMatch') : t('market.empty')}</p>}
          <small className='vpet-muted'>{t('market.count', rows.length)}</small>
        </section>
        <aside className='vpet-market__stall' aria-label={t('market.post')}>
          <h4>{t('market.post')}</h4>
          {chosen && <small>{t('market.maxPrice', maxPrice)}</small>}
          <div className='vpet-market__field'>
            {t('market.itemToSell')}
            <ItemPicker items={stock} state={state} value={chosen?.id || ''} busy={busy} label={t('market.itemToSell')}
              onPick={(id) => { setItem(id); setQty(1); }}/>
          </div>
          {chosen && (
            <div className={`vpet-market__preview is-${RARITY[chosen.rarity || 0]}`}>
              <ItemIcon id={chosen.id} size={44}/>
              <div><b>{chosen.name}</b><small>{chosen.desc}</small></div>
            </div>
          )}
          <label className='vpet-market__field'>
            {t('market.qty')}
            <input aria-label={t('market.qty')} type='number' min='1' max={have || 1} value={qty} disabled={busy || !chosen} onChange={(e) => setQty(e.target.value)}/>
          </label>
          <label className='vpet-market__field'>
            {t('market.lotPrice')}
            <input aria-label={t('market.lotPrice')} type='number' min='1' value={price} placeholder={t('market.priceHint')} disabled={busy || !chosen} onChange={(e) => setPrice(e.target.value)}/>
          </label>
          {asking > 0 && (
            <dl className='vpet-fusion__cost'>
              <div><dt>{t('market.youGet')}</dt><dd>{`${earns} 🪙`}</dd></div>
              <div><dt>{t('market.tax', tax)}</dt><dd>{`${asking - earns} 🪙`}</dd></div>
              <div><dt>{t('market.fee')}</dt><dd>{`${fee} 🪙`}</dd></div>
            </dl>
          )}
          {blocked && <p className='vpet-inv__warning'>{blocked}</p>}
          {confirm?.kind === 'sell' ? <div className='vpet-market__confirm'>
            <p>{t('market.sellLine', lot, chosen?.name, fee)}</p>
            <button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm(null)}>{t('app.cancel')}</button>
            <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(blocked)} onClick={sell}>{t('market.confirmSell')}</button>
          </div> : <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(blocked)} onClick={() => setConfirm({kind: 'sell'})}>{busy ? t('market.posting') : t('market.post')}</button>}
          <h4>{t('market.mine', mine.length)}</h4>
          {mine.length ? (
            <ul className='vpet-market__list is-mine'>
              {mine.map((r) => <Row key={r.id} row={r} state={state} busy={busy} onBuy={(id) => setConfirm({kind: 'buy', id})} onCancel={cancel}/>)}
            </ul>
          ) : <p className='vpet-muted'>{t('market.noneListed')}</p>}
          {children}
        </aside>
      </div>
      )}
      {view === 'want' && (
        <div className='vpet-market__body'>
          <section className='vpet-market__board' aria-label={t('market.wantBoard')}>
            <div className='vpet-market__tools'>
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => refresh()}>{t('app.refresh')}</button>
              <label className='vpet-inv__search-label'>{t('market.searchWant')}<input className='vpet-inv__search' type='search' placeholder={t('market.searchWantHint')} value={query} onChange={(e) => setQuery(e.target.value)}/></label>
            </div>
            {orders.length ? <ul className='vpet-market__list'>{orders.map((r) => <WantRow key={r.id} row={r} state={state} busy={busy} onFulfill={fulfill} onCancel={cancelWant}/>)}</ul>
              : <p className='vpet-empty'>{query ? t('market.wantNoMatch') : t('market.wantEmpty')}</p>}
            <small className='vpet-muted'>{t('market.wantCount', orders.length)}</small>
          </section>
          <aside className='vpet-market__stall' aria-label={t('market.postWant')}>
            <h4>{t('market.postWant')}</h4>
            <small>{t('market.wantHint')}</small>
            <div className='vpet-market__field'>
              {t('market.itemWanted')}
              <RequestPicker items={requestable} state={state} value={wanted?.id || ''} busy={busy}
                onPick={(id) => { setWantItem(id); setWantQty(1); }}/>
            </div>
            <label className='vpet-market__field'>{t('market.qtyWanted')}<input aria-label={t('market.qtyWanted')} type='number' min='1' max='99' value={wantQty} disabled={busy} onChange={(e) => setWantQty(e.target.value)}/></label>
            <label className='vpet-market__field'>{t('market.totalPrice')}<input aria-label={t('market.totalPrice')} type='number' min='1' value={wantPrice} placeholder={t('market.priceHint')} disabled={busy} onChange={(e) => setWantPrice(e.target.value)}/></label>
            {bid > 0 && <dl className='vpet-fusion__cost'><div><dt>{t('market.heldUp')}</dt><dd>{`${bid} 🪙`}</dd></div><div><dt>{t('market.fee')}</dt><dd>{`${fee} 🪙`}</dd></div><div><dt>{t('market.unitPrice')}</dt><dd>{`${Math.round(bid / requestedQty)} 🪙`}</dd></div></dl>}
            {wantBlocked && <p className='vpet-inv__warning'>{wantBlocked}</p>}
            {confirm?.kind === 'want' ? <div className='vpet-market__confirm'><p>{t('market.wantLine', bid, requestedQty, wanted?.name, fee)}</p><button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm(null)}>{t('app.cancel')}</button><button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(wantBlocked)} onClick={postWant}>{t('market.confirmWant')}</button></div>
              : <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(wantBlocked)} onClick={() => setConfirm({kind:'want'})}>{t('market.postWant')}</button>}
            <h4>{t('market.myWants', myOrders.length)}</h4>
            {myOrders.length ? <ul className='vpet-market__list is-mine'>{myOrders.map((r) => <WantRow key={r.id} row={r} state={state} busy={busy} onFulfill={fulfill} onCancel={cancelWant}/>)}</ul> : <p className='vpet-muted'>{t('market.noWants')}</p>}
            {children}
          </aside>
        </div>
      )}
    </div>
  );
}
