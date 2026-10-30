import React, {useEffect, useRef, useState} from 'react';

import * as api from './api';
import {ACTIONS} from './actions.cjs';
import Boundary from './Boundary';
import {PHONE_TABS, TABS, openingTab, readTab, writeTab} from './tabs.cjs';
import City from './City';
import Creator from './Creator';
import DuelTab from './Duel';
import GameIcon from './GameIcon';
import Gift from './Gift';
import HuntTab, {Badges, GearBox, Skills, RARITY} from './Hunt';
import Inventory from './Inventory';
import Quests, {Gems} from './Quests';
import Clinic, {StatSheet} from './Clinic';
import Checkin from './Checkin';
import {dateLocale, t, tr} from './i18n';
import ItemIcon from './ItemIcon';
import MarketTab from './Market';
import PetCanvas from './PetCanvas';
import ShopTab, {Coins} from './Shop';
import Campaign from './Campaign';
import {clearJump, itemOf, run, select, useVPet} from './store';
import WorldChat from './WorldChat';

const EFFECT_KEYS = {cay: 'pet.spicy', hoi: 'pet.stinky', ngon: 'pet.tasty', hyper: 'pet.hyper', ngodoc: 'pet.poisoned', dau: 'pet.angry', cuoi: 'pet.giggling', ngu: 'pet.sleeping'};
// Each care group has one obvious source, so an empty group points at it rather than
// leaving the player to guess.
const EMPTY_SUPPLY = {
  food: {text: 'pet.emptyFood', action: 'pet.goShop', tab: 'shop'},
  med: {text: 'pet.emptyMed', action: 'pet.goClinic', tab: 'clinic'},
  energy: {text: 'pet.emptyEnergy', action: 'pet.goShop', tab: 'shop'},
  acc: {text: 'pet.emptyAcc', action: 'pet.goMarket', tab: 'market'},
  gear: {text: 'pet.emptyGear', action: 'pet.goHunt', tab: 'hunt'},
};

const KIND_KEYS = {food: 'kind.food', med: 'kind.med', energy: 'kind.energy', acc: 'kind.acc', card: 'kind.card', gear: 'kind.gear', bag: 'kind.bag', stone: 'kind.stone', revive: 'kind.revive', junk: 'kind.junk'};
// Reopening the game returns to the screen last used; openingTab decides what is safe to
// reopen, and the rules are tested in tabs.test.cjs.

function age(born, now) {
  const h = Math.floor((now - born) / 3.6e6);
  return h < 24 ? t('pet.ageHours', h) : t('pet.ageDays', Math.floor(h / 24));
}

function Bar({label, value, color}) {
  const v = Math.round(value);
  return (
    <div className='vpet-bar'>
      <span>{label}</span>
      <div className='vpet-bar__track' role='meter' aria-label={label} aria-valuenow={v} aria-valuemin={0} aria-valuemax={100}>
        <div className={`vpet-bar__fill${v < 25 ? ' is-low' : ''}`} style={{width: `${v}%`, background: color}}/>
      </div>
      <b>{v}</b>
    </div>
  );
}

// Items show their effect behind the ℹ️ corner, the way an RPG bag does.
export function ItemGrid({inv, filter, onUse, busy, empty, onEmptyAction, emptyAction, actionLabel = t('pet.use')}) {
  const {state} = useVPet();
  const [info, setInfo] = useState('');
  const items = (state?.catalog || []).filter((it) => inv[it.id] > 0 && (!filter || filter(it)));
  if (!items.length) {
    return (
      <div className='vpet-items__empty'>
        <p>{empty || t('pet.emptyBag')}</p>
        {emptyAction && <button type='button' className='vpet-btn vpet-btn--primary' onClick={onEmptyAction}>{emptyAction}</button>}
      </div>
    );
  }
  const shown = items.find((it) => it.id === info);
  return (
    <>
      <div className='vpet-items'>
        {items.map((it) => (
          <div key={it.id} className={`vpet-item is-${RARITY[it.rarity || 0]}`}>
            <button type='button' className='vpet-item__use' disabled={busy} onClick={() => onUse(it.id)} title={`${actionLabel} ${it.name}`} aria-label={`${actionLabel} ${it.name}`}>
              <ItemIcon id={it.id} size={34}/>
              <span className='vpet-item__name'>{it.name}</span>
            </button>
            <span className='vpet-item__count'>{`×${inv[it.id]}`}</span>
            <button type='button' className='vpet-item__info' aria-label={t('pet.itemEffect', it.name)} onClick={() => setInfo(info === it.id ? '' : it.id)}>{'ℹ️'}</button>
          </div>
        ))}
      </div>
      {shown && (
        <p className={`vpet-tip is-${RARITY[shown.rarity || 0]}`} role='status'>
          <b><ItemIcon id={shown.id} size={22}/>{shown.name}</b>
          {` — ${shown.desc}`}
          {KIND_KEYS[shown.kind] && <small>{t(KIND_KEYS[shown.kind])}</small>}
        </p>
      )}
    </>
  );
}

function Switcher({state, sel, onCreate}) {
  const free = state.pets.length < state.slots;
  if (state.pets.length < 2 && !free) {
    return null;
  }
  return (
    <div className='vpet-switch' role='tablist' aria-label={t('pet.select')}>
      {state.pets.map((p, i) => (
        <button key={`${i}-${p.born}`} type='button' role='tab' aria-selected={i === sel} className={i === sel ? 'is-active' : ''} onClick={() => select(i)}>
          <PetCanvas pet={p} parts={p.parts} scale={1} still={true}/>
          {p.dead ? `🪦 ${p.name}` : p.name}
        </button>
      ))}
      {free && state.pets.length > 0 && <button type='button' onClick={onCreate}>{t('pet.hatchMore', state.pets.length, state.slots)}</button>}
    </div>
  );
}

// Renaming is its own small form: the card keeps the pet's looks and record, so it does not
// belong in the reskin flow where every part is up for change.
function Build({state, sel, busy, act}) {
  const pet = state.pets[sel];
  const left = state.build_points?.[sel] || 0;
  const total = (pet.build?.atk || 0) + (pet.build?.def || 0) + (pet.build?.hp || 0);
  const stats = state.stats?.[sel];
  // One line per stat, the way an RPG character sheet reads: name, the combat number the
  // points feed, the points spent, and the button. No stacked cards, no per-row paragraph.
  const rows = [
    ['atk', t('pet.attack'), t('pet.damage'), stats?.atk, '+0.8', '#df6b57'],
    ['def', t('pet.defence'), t('pet.armour'), stats?.def, '+0.6', '#4c91d8'],
    ['hp', t('pet.vitality'), t('pet.battleHP'), stats?.hp, '+4', '#53aa72'],
  ];
  const spend = (id) => { if (!busy && left && !pet.dead) act(() => api.allocate(sel, id)); };
  return (
    <section className='vpet-build' aria-label={t('pet.buildLabel')}>
      <header>
        <div>
          <b>{t('pet.build')}</b>
          <small>{`Lv ${pet.lv || 1} · đã cộng ${total}${pet.build_bonus ? ` · thưởng ${pet.build_bonus}` : ''}`}</small>
        </div>
        <strong className={left ? 'is-ready' : ''}>{left}<small>{t('pet.points')}</small></strong>
      </header>
      <ul className='vpet-build__list'>
        {rows.map(([id, name, label, value, per, color]) => (
          <li key={id} style={{'--stat': color}}>
            <span className='vpet-build__name'><b>{name}</b><small>{label}</small></span>
            <b className='vpet-build__value'>{value === undefined ? '—' : Math.round(value * 10) / 10}</b>
            <span className='vpet-build__spent'>{`${pet.build?.[id] || 0} điểm`}</span>
            <button type='button' aria-label={t('pet.addStat', name)} title={t('pet.perPoint', per, label)}
              disabled={busy || !left || pet.dead} onClick={() => spend(id)}>{'+'}</button>
          </li>
        ))}
      </ul>
      <p className='vpet-muted'>{t('pet.buildHint')}</p>
    </section>
  );
}

function RenameBox({state, sel, busy, act, onDone}) {
  const pet = state.pets[sel];
  const [name, setName] = useState(pet.name);
  const bad = !name.trim() ? t('pet.renameEmpty') : name.trim() === pet.name ? t('pet.renameSame') : [...name.trim()].length > 20 ? t('pet.renameLong') : '';
  return (
    <div className='vpet-rename'>
      <label className='vpet-market__field'>
        {t('pet.renameFor', pet.name)}
        <input type='text' maxLength={40} value={name} autoFocus={true} disabled={busy} onChange={(e) => setName(e.target.value)}/>
      </label>
      <p className='vpet-muted'>{t('pet.renameCost', state.inv.the_ten || 0)}</p>
      {bad && <p className='vpet-inv__warning'>{bad}</p>}
      <div className='vpet-rename__btns'>
        <button type='button' className='vpet-btn' disabled={busy} onClick={onDone}>{t('app.cancel')}</button>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(bad)} onClick={() => act(() => api.renamePet(sel, name.trim())).then((r) => r && onDone())}>{t('pet.renameAction')}</button>
      </div>
    </div>
  );
}

function PetTab({state, sel, busy, act, onCreate, onReskin, onInventory, onGo}) {
  const [supply, setSupply] = useState('food');
  const [renaming, setRenaming] = useState(false);
  const [sheetTab, setSheetTab] = useState('skills');
  const pet = state.pets[sel];
  if (!pet) {
    return <Creator/>;
  }
  if (pet.dead) {
    return (
      <div className='vpet-dead'>
        <PetCanvas pet={pet} parts={pet.parts} scale={5}/>
        <h3>{t('pet.deadTitle', pet.name)}</h3>
        <p>{t('pet.deadCause', pet.cause, age(pet.born, pet.died_at || state.now))}</p>
        {state.inv.long > 0 && <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.useItem(sel, 'long'))}>{t('pet.reviveFeather')}</button>}
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy} onClick={() => act(() => api.bury(sel))}>{t('pet.bury')}</button>
      </div>
    );
  }
  const known = Object.entries(pet.known || {});
  const care = ACTIONS.filter(([id]) => !['bung', 'dam'].includes(id));
  return (
    <div className='vpet-pettab'>
      <div className='vpet-sheet'>
        <section className='vpet-sheet__hero'>
          <header className='vpet-character__head'>
            <div><h3>{pet.name}</h3><p className='vpet-muted'>{t('pet.ageFed', age(pet.born, state.now), pet.fed)}</p></div>
            <span className='vpet-character__gearhint'>{t('pet.gearHint')}</span>
          </header>
          <div className='vpet-tags'>
            {pet.sick && <span className='vpet-tag vpet-tag--bad'>{t('pet.sickTag')}</span>}
            {pet.effect && <span className='vpet-tag'>{t(EFFECT_KEYS[pet.effect])}</span>}
            {pet.frozen_until > state.now && <span className='vpet-tag'>{t('pet.vacationUntil', new Date(pet.frozen_until).toLocaleString(dateLocale()))}</span>}
            {pet.care_until > state.now && <span className='vpet-tag'>{t('pet.careUntil', new Date(pet.care_until).toLocaleString(dateLocale()))}</span>}
            {pet.acc && <button type='button' className='vpet-tag' disabled={busy} onClick={() => act(() => api.unequip(sel))} title={t('pet.removeAcc')}>{`${itemOf(pet.acc)?.name || ''} ×`}</button>}
          </div>
          <GearBox state={state} sel={sel} busy={busy} act={act} onInventory={onInventory}>
            <button type='button' className='vpet-pettab__pet' disabled={busy} onClick={() => act(() => api.interact(sel, 'vuot'))} title={t('pet.stroke')}>
              <PetCanvas pet={pet} parts={pet.parts} scale={4}/>
            </button>
          </GearBox>
          <Badges state={state} sel={sel}/>
        </section>
        <section className='vpet-sheet__care'>
          <header className='vpet-section-head'><h4>{t('pet.care')}</h4><span className='vpet-muted'>{pet.sick ? t('pet.needCure') : pet.health < 40 ? t('pet.needRecovery') : pet.hunger < 25 ? t('pet.needFood') : t('pet.ready')}</span></header>
          <div className='vpet-vitals'>
            <Bar label={t('pet.health')} value={pet.health} color='#39b976'/>
            <Bar label={t('pet.hunger')} value={pet.hunger} color='#e5a243'/>
            <Bar label={t('pet.happy')} value={pet.happy} color='#a587df'/>
          </div>
          {state.energy?.[sel] && (
            <p className='vpet-energy-line' title={state.energy[sel].now >= state.energy[sel].max ? t('pet.fullEnergy') : `Hồi 1 điểm mỗi ${Math.round(state.energy[sel].tick_seconds/60)} phút`}>
              <b>{`⚡ ${state.energy[sel].now}/${state.energy[sel].max}`}</b>
              <span>{state.energy[sel].now >= state.energy[sel].max ? t('pet.full') : `+1 / ${Math.round(state.energy[sel].tick_seconds/60)} phút`}</span>
            </p>
          )}
          <div className='vpet-actions'>
            {care.map(([id, icon, label]) => <button key={id} type='button' disabled={busy} onClick={() => act(() => api.interact(sel, id))}>{t(label)}</button>)}
            {state.inv.the_skin > 0 && <button type='button' disabled={busy} onClick={onReskin}>{t('pet.reskin', state.inv.the_skin)}</button>}
            {state.inv.the_ten > 0 && <button type='button' disabled={busy} onClick={() => setRenaming(true)}>{t('pet.rename', state.inv.the_ten)}</button>}
          </div>
          {renaming && <RenameBox state={state} sel={sel} busy={busy} act={act} onDone={() => setRenaming(false)}/>}
          <section className='vpet-sheet__use'>
            <header className='vpet-section-head'>
              <div className='vpet-supply-tabs' role='group' aria-label={t('pet.supplies')}>
                {[['food', t('pet.food')], ['med', t('pet.recovery')], ['energy', t('pet.energy')], ['acc', t('pet.accessory')], ['gear', t('pet.gear')]].map(([id, label]) => <button key={id} type='button' aria-pressed={supply === id} onClick={() => setSupply(id)}>{t(label)}</button>)}
              </div>
              <button type='button' className='vpet-textlink' onClick={onInventory}>{t('pet.openBag')}</button>
            </header>
            {supply === 'acc' && pet.acc && (
              <div className='vpet-worn-acc'>
                <ItemIcon id={pet.acc} size={26}/>
                <span>{t('pet.wearing', itemOf(pet.acc)?.name || pet.acc)}</span>
                <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.unequip(sel))}>{t('pet.remove')}</button>
              </div>
            )}
            <ItemGrid
              inv={state.inv}
              filter={(it) => it.kind === supply}
              busy={busy}
              empty={t(EMPTY_SUPPLY[supply]?.text)}
              emptyAction={t(EMPTY_SUPPLY[supply]?.action)}
              onEmptyAction={() => onGo(EMPTY_SUPPLY[supply]?.tab)}
              actionLabel={supply === 'gear' ? t('pet.equip') : t('pet.use')}
              onUse={(id) => act(() => supply === 'gear' ? api.equip(sel, id) : api.useItem(sel, id))}
            />
          </section>
        </section>
        <section className='vpet-sheet__gear'>
          <nav className='vpet-sheet-tabs' role='tablist' aria-label={t('pet.development')}>
            <button type='button' role='tab' aria-selected={sheetTab === 'skills'} onClick={() => setSheetTab('skills')}>{t('pet.skills')}</button>
            <button type='button' role='tab' aria-selected={sheetTab === 'build'} onClick={() => setSheetTab('build')}>{`${t('pet.build')} (${state.build_points?.[sel] || 0})`}</button>
            <button type='button' role='tab' aria-selected={sheetTab === 'stats'} onClick={() => setSheetTab('stats')}>{t('pet.stats')}</button>
          </nav>
          {sheetTab === 'skills' ? <><header className='vpet-section-head'><h4>{t('pet.combatSkills')}</h4><span className='vpet-muted'>{t('pet.skillStones', state.inv.ngoc || 0)}</span></header><Skills state={state} sel={sel} busy={busy} act={act}/></> : sheetTab === 'build' ? <Build state={state} sel={sel} busy={busy} act={act}/> : <StatSheet state={state} sel={sel}/>}
        </section>
        <section className='vpet-sheet__more'>
          <details>
            <summary>{t('pet.more')}</summary>
            <div className='vpet-actions'>
              {ACTIONS.filter(([id]) => ['bung', 'dam'].includes(id)).map(([id, icon, label]) => <button key={id} type='button' disabled={busy} onClick={() => act(() => api.interact(sel, id))}>{t(label)}</button>)}
            </div>
            {known.length > 0 && <p className='vpet-muted'>{known.map(([id, v]) => `${v === 'like' ? t('pet.like') : t('pet.hate')}: ${itemOf(id)?.name || id}`).join(' · ')}</p>}
            <Cull pet={pet} sel={sel} busy={busy} act={act}/>
          </details>
        </section>
      </div>
    </div>
  );
}

// Putting a pet down is deliberate: it takes two clicks and states the penalty first.
function Cull({pet, sel, busy, act}) {
  const [armed, setArmed] = useState(false);
  if (!armed) {
    return <button type='button' className='vpet-btn vpet-btn--ghost vpet-cull' disabled={busy} onClick={() => setArmed(true)}>{'🔪 Giết thú'}</button>;
  }
  return (
    <div className='vpet-cull__warn'>
      <p>{`Giết ${pet.name} sẽ gắn nhãn ☠️ Kẻ tàn bạo trong 15 ngày: không ấp được thú mới, thú còn lại mau đói và buồn, thuốc kém tác dụng, tiền thưởng giảm nửa.`}</p>
      <button type='button' className='vpet-btn' disabled={busy} onClick={() => setArmed(false)}>{tr('Thôi','Cancel','やめる')}</button>
      <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy} onClick={() => act(() => api.kill(sel)).then(() => setArmed(false))}>{tr('Vẫn giết','Kill anyway','それでも殺す')}</button>
    </div>
  );
}

const bundle = (m) => Object.entries(m || {}).map(([id, n]) => `${n}× ${itemOf(id)?.emoji || ''} ${itemOf(id)?.name || id}`).join(', ') || tr('(không có)','(none)','（なし）');

// 1.14 replaced barter with the market. Offers already in flight still hold escrowed items,
// so this screen stays as a drain: answer or cancel what is open, but no new offers.
function LegacyOffers({state, busy, act}) {
  const open = [...(state.in || []), ...(state.out || [])];
  if (!open.length) return null;
  return (
    <details className='vpet-trade vpet-legacy-offers'>
      <summary>{`Trao đổi kiểu cũ đang chờ (${open.length})`}</summary>
      <p className='vpet-muted'>{'Chợ đã thay cho trao đổi 1-1. Trả lời hoặc hủy để lấy lại đồ đang giữ.'}</p>
      {(state.in || []).map((o) => (
        <div key={o.id} className='vpet-offer'>
          <p>{`${o.from_name} tặng: ${bundle(o.give)}`}<br/>{`Muốn nhận: ${bundle(o.want)}`}</p>
          <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy} onClick={() => act(() => api.offerAction(o.id, 'accept'))}>{'Đồng ý'}</button>
          <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.offerAction(o.id, 'decline'))}>{'Từ chối'}</button>
        </div>
      ))}
      {(state.out || []).map((o) => (
        <div key={o.id} className='vpet-offer'>
          <p>{`Gửi ${o.to_name}: tặng ${bundle(o.give)}, đổi lấy ${bundle(o.want)}`}</p>
          <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.offerAction(o.id, 'cancel'))}>{'Hủy và lấy lại đồ'}</button>
        </div>
      ))}
    </details>
  );
}

function GraveTab({state}) {
  if (!state.graves?.length) {
    return <p className='vpet-empty'>{tr('Nghĩa trang trống. Chăm thú thật tốt nhé!','The graveyard is empty. Take good care of your pet!','墓地は空です。大切に育ててください！')}</p>;
  }
  return (
    <div className='vpet-graves'>
      {state.graves.map((g) => (
        <div key={`${g.name}-${g.died}`} className='vpet-grave'>
          <PetCanvas parts={g.parts} pet={{}} scale={2} still={true}/>
          <div>
            <b>{g.name}</b>
            <p className='vpet-muted'>{`Sống ${age(g.born, g.died)} · ${g.cause}`}</p>
            <p className='vpet-muted'>{new Date(g.died).toLocaleString('vi-VN')}</p>
          </div>
        </div>
      ))}
    </div>
  );
}

// One tab button, used by the bar and by the phone's "More" sheet.
function TabButton({id, badge, active, className = '', onGo}) {
  const label = t(`tab.${id}`);
  return (
    <button type='button' title={label} aria-label={label} aria-current={active ? 'page' : undefined}
      className={`${className}${active ? ' is-active' : ''}`} onClick={() => onGo(id)}>
      <GameIcon name={id}/>
      <span className='vpet-tabs__label'>{label}</span>
      {badge > 0 && <span className='vpet-badge'>{badge}</span>}
    </button>
  );
}

export default function Panel({account, profile}) {
  const {state, bubble, sel, jump} = useVPet();
  const [tab, setTab] = useState(() => openingTab(readTab(globalThis.localStorage || {})));
  const [creating, setCreating] = useState(false); // false, 'new' or 'reskin'
  const [busy, setBusy] = useState(false);
  const [replay, setReplay] = useState('');
  // an item picked in the bag opens the market with its sell form already filled in
  const [sellItem, setSellItem] = useState('');
  const [settings, setSettings] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  // the calendar shows itself once a day; dismissing it must not bring it back this session
  const [checkinOpen, setCheckinOpen] = useState(false);
  const checkinSeen = useRef(false);
  const [giftOpen, setGiftOpen] = useState(false);
  const giftSeen = useRef(false);
  useEffect(() => {
    if (tab !== 'city' || giftSeen.current || !state?.welcome_gift || state.welcome_gift.claimed) return;
    giftSeen.current = true;
    setCheckinOpen(false);
    setGiftOpen(true);
  }, [tab, state?.welcome_gift?.claimed]);
  useEffect(() => {
    // The one-time gift has priority, so two automatic dialogs never overlap.
    if (tab !== 'city' || giftOpen || (state?.welcome_gift && !state.welcome_gift.claimed) || checkinSeen.current || !state?.checkin || state.checkin.claimed_today) return;
    checkinSeen.current = true;
    setCheckinOpen(true);
  }, [tab, giftOpen, state?.welcome_gift?.claimed, state?.checkin?.claimed_today]);
  useEffect(() => {
    if (jump) {
      const asked = jump.tab === 'trade' ? 'market' : jump.tab;
      if (TABS.includes(asked)) {
        setTab(asked);
        writeTab(globalThis.localStorage || {}, asked);
      }
      setCreating(false);
      setReplay(jump.duel || '');
      clearJump();
    }
  }, [jump]);
  useEffect(() => {
    if (!moreOpen) return undefined;
    const close = (e) => { if (e.key === 'Escape') setMoreOpen(false); };
    document.addEventListener('keydown', close);
    return () => document.removeEventListener('keydown', close);
  }, [moreOpen]);
  const act = async (call, quiet = false) => {
    setBusy(true);
    try {
      return await run(call, undefined, quiet);
    } catch (e) {
      // shown in the bubble
      return null;
    } finally {
      setBusy(false);
    }
  };
  // A remembered tab is only as good as the data behind it: burying the last pet has to move
  // the player off a screen that needs one, not just at startup but the moment it happens.
  useEffect(() => {
    if (state) {
      setTab((current) => openingTab(current, state.pets?.length || 0));
    }
  }, [state?.pets?.length]);

  const badges = {
    market: state?.in?.length || 0,
    duel: state?.din?.length || 0,
    quest: (state?.quests || []).filter((q) => q.have >= q.need && !q.done).length,
  };
  const overflow = TABS.slice(PHONE_TABS);
  const overflowBadges = overflow.reduce((sum, id) => sum + (badges[id] || 0), 0);
  const overflowActive = overflow.includes(tab);
  const go = (id) => {
    setCheckinOpen(false);
    setGiftOpen(false);
    setMoreOpen(false);
    setCreating(false);
    setTab(id);
    writeTab(globalThis.localStorage || {}, id);
  };

  return (
    <main className='vpet-world-shell'>
      <div className='vpet-panel'>
        <header className='vpet-panel__head'>
          <div className='vpet-panel__title'><h2>{t('app.title')}</h2><span>{t(`tab.${tab}`)}</span></div>
          {state && <Coins state={state}/>}
          {state?.air > 0 && (
            <button type='button' className='vpet-iconbtn vpet-airdrop' disabled={busy} aria-label={t('buddy.crate')} title={t('buddy.crate')} onClick={() => act(api.claimAirdrop)}>
              {'📦'}<span className='vpet-badge'>{state.air}</span>
            </button>
          )}
          {state?.welcome_gift && !state.welcome_gift.claimed && (
            <button type='button' className='vpet-iconbtn vpet-gift__open' aria-label={t('gift.open')} title={t('gift.open')} onClick={() => { setCheckinOpen(false); setGiftOpen(true); }}>{'🎁'}</button>
          )}
          {state?.checkin && (
            <button type='button' className={`vpet-iconbtn vpet-checkin__open${state.checkin.claimed_today ? '' : ' is-due'}`}
              aria-label={state.checkin.claimed_today ? t('checkin.openDone') : t('checkin.open')}
              title={state.checkin.claimed_today ? t('checkin.streakNow', state.checkin.streak) : t('checkin.open')}
              onClick={() => setCheckinOpen(true)}>{'📅'}</button>
          )}
          {state && (
            <div className='vpet-gear-menu'>
              <button type='button' className='vpet-iconbtn' aria-label={t('app.settings')} aria-expanded={settings} onClick={() => setSettings(!settings)}><GameIcon name='settings'/></button>
              {settings && (
                <div className='vpet-setmenu' role='menu'>
                  <label className='vpet-mute'>
                    <input type='checkbox' checked={!state.muted} onChange={(e) => act(() => api.setMuted(!e.target.checked))}/>
                    {t('app.notify')}
                  </label>
                </div>
              )}
            </div>
          )}
          {profile}
        </header>
        <nav className='vpet-tabs' aria-label='Chức năng game'>
          {TABS.map((id, i) => (
            <TabButton key={id} id={id} badge={badges[id] || 0} active={tab === id}
              className={i >= PHONE_TABS ? 'is-overflow' : ''} onGo={go}/>
          ))}
          <button type='button' className={`vpet-tabs__more${overflowActive ? ' is-active' : ''}`}
            aria-label={t('tab.more')} aria-expanded={moreOpen} onClick={() => setMoreOpen(true)}>
            <GameIcon name='more'/>
            <span className='vpet-tabs__label'>{t('tab.more')}</span>
            {overflowBadges > 0 && <span className='vpet-badge'>{overflowBadges}</span>}
          </button>
        </nav>
        {moreOpen && (
          <div className='vpet-more'>
            <button type='button' className='vpet-more__backdrop' aria-label={t('app.close')} onClick={() => setMoreOpen(false)}/>
            <div className='vpet-more__sheet' role='dialog' aria-modal='true' aria-label={t('tab.more')}>
              <h3>{t('tab.more')}</h3>
              <div className='vpet-more__grid'>
                {TABS.slice(PHONE_TABS).map((id) => (
                  <TabButton key={id} id={id} badge={badges[id] || 0} active={tab === id} onGo={go}/>
                ))}
              </div>
            </div>
          </div>
        )}
        {state?.cruel_days > 0 && (
          <div className='vpet-cruel' role='status'>
            <b>{'☠️ Kẻ tàn bạo'}</b>
            <span>{`Còn ${state.cruel_days} ngày. Không ấp được thú mới · thú mau đói và buồn · thuốc kém tác dụng · tiền thưởng giảm nửa.`}</span>
          </div>
        )}
        {giftOpen && state && tab === 'city' && <Gift state={state} busy={busy} act={act} onClose={() => setGiftOpen(false)}/>}
        {checkinOpen && !giftOpen && state && tab === 'city' && <Checkin state={state} busy={busy} act={act} onClose={() => setCheckinOpen(false)}/>}
        {bubble && <div className='vpet-toast' role='status'>{bubble.text}</div>}
        <div className={`vpet-panel__body${tab === 'city' ? ' is-city' : ''}`}>
          <Boundary resetKey={tab} onHome={() => go('city')}>
          {!state && <p className='vpet-empty'>{t('app.loading')}</p>}
          {state && !creating && tab === 'city' && state.pets.length > 0 && (
            <div className='vpet-home'>
              <City state={state} sel={sel} onGo={go}/>
              <WorldChat account={account}/>
            </div>
          )}
          {state && !creating && tab === 'city' && state.pets.length === 0 && <Creator/>}
          {state && !creating && ['pet', 'hunt', 'world', 'duel'].includes(tab) && <Switcher state={state} sel={sel} onCreate={() => setCreating('new')}/>}
          {state && creating === 'new' && (
            <Creator
              onDone={() => {
                setCreating(false);
                select(state.pets.length);
              }}
            />
          )}
          {state && creating === 'reskin' && state.pets[sel] && <Creator key={sel} reskin={sel} pet={state.pets[sel]} onDone={() => setCreating(false)}/>}
          {state && !creating && tab === 'pet' && <PetTab state={state} sel={sel} busy={busy} act={act} onCreate={() => setCreating('new')} onReskin={() => setCreating('reskin')} onInventory={() => go('bag')} onGo={go}/>}
          {state && !creating && tab === 'world' && <Campaign state={state} sel={sel} busy={busy} act={act}/>}
          {state && !creating && tab === 'hunt' && <HuntTab state={state} sel={sel} busy={busy} act={act}/>}
          {state && !creating && tab === 'shop' && <ShopTab state={state} busy={busy} act={act}/>}
          {state && !creating && tab === 'bag' && <Inventory state={state} sel={sel} busy={busy} act={act} onSkills={() => go('pet')} onReskin={() => { go('pet'); setCreating('reskin'); }} onSell={(id) => { setSellItem(id); go('market'); }}/>}
          {state && !creating && tab === 'market' && (
            <MarketTab state={state} busy={busy} act={act} sellItem={sellItem} onSellItemUsed={() => setSellItem('')}>
              <LegacyOffers state={state} busy={busy} act={act}/>
            </MarketTab>
          )}
          {state && !creating && tab === 'duel' && <DuelTab state={state} sel={sel} busy={busy} act={act} replay={replay} onReplayed={() => setReplay('')}/>}
          {state && !creating && tab === 'quest' && <Quests state={state} sel={sel} busy={busy} act={act}/>}
          {state && !creating && tab === 'clinic' && <Clinic state={state} sel={sel} busy={busy} act={act}/>}
          {state && !creating && tab === 'grave' && <GraveTab state={state}/>}
          </Boundary>
        </div>
      </div>
    </main>
  );
}
