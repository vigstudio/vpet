import React, {useState} from 'react';

import * as api from './api';
import {dateLocale, t, tr} from './i18n';
import HuntScene, {Backdrop as Scene} from './HuntScene';
import ItemIcon from './ItemIcon';
import PetCanvas from './PetCanvas';
import Realm, {REALMS} from './Realm';
import {Parties} from './Campaign';
import {itemOf, useNow} from './store';

export const RARITY = ['r0', 'r1', 'r2', 'r3', 'r4', 'r5', 'r6'];


export function xpNeed(lv) {
  return 20 + (lv * 12);
}

export function Rarity({id, children}) {
  const r = itemOf(id)?.rarity || 0;
  return <span className={`vpet-rar is-${RARITY[r]}`}>{children}</span>;
}

// What the pet brought home, shown until the user closes it.
function Report({state, hunt, onClose}) {
  const map = (state.maps || []).find((m) => m.id === hunt.map);
  return (
    <div className='vpet-report'>
      <h4>{`${map?.emoji || '🗺️'} ${map?.name || t('hunt.trip')}`}</h4>
      <div className='vpet-report__loot'>
        {hunt.finds.map((f, i) => (
          <span key={`${f.item}-${i}`} className={`vpet-loot is-${RARITY[f.rarity]}`}>
            <ItemIcon id={f.item} size={32}/>
            {itemOf(f.item)?.name || f.item}
            <small>{(state.rarities || [])[f.rarity]}</small>
          </span>
        ))}
      </div>
      <p>{tr(`✨ +${hunt.xp} kinh nghiệm`, `✨ +${hunt.xp} XP`, `✨ 経験値 +${hunt.xp}`)}{hunt.hurt > 0 && tr(` · 🩸 mất ${Math.round(hunt.hurt)} sức khỏe`, ` · 🩸 lost ${Math.round(hunt.hurt)} health`, ` · 🩸 体力を${Math.round(hunt.hurt)}失う`)}</p>
      {hunt.note && <p className='vpet-report__note'>{hunt.note}</p>}
      <button type='button' className='vpet-btn vpet-btn--primary' onClick={onClose}>{t('hunt.continue')}</button>
    </div>
  );
}

export default function HuntTab({state, sel, busy, act}) {
  const [report, setReport] = useState(null);
  const [scene, setScene] = useState(null);
  const [pick,setPick]=useState('vuon');
  const [team,setTeam]=useState(false);
  // hooks stay above every early return, or React tears the component down mid-render
  const now = useNow();
  const pet = state.pets[sel];
  if (!pet) {
    return <p className='vpet-empty'>{t('hunt.noPet')}</p>;
  }
  if (scene) {
    const show = () => { setReport(scene); setScene(null); };
    return (
      <HuntScene
        hunt={scene}
        map={(state.maps || []).find((m) => m.id === scene.map)}
        onDone={show}
        onSkip={show}
      />
    );
  }
  if (report) {
    return <Report state={state} hunt={report} onClose={() => setReport(null)}/>;
  }
  const lv = pet.lv || 1;
  const left = Math.max(0, Math.ceil((((pet.last_hunt || 0) + (10 * 60000)) - now) / 1000));
  const rest = left > 0 ? `${Math.floor(left / 60)}:${String(left % 60).padStart(2, '0')}` : '';
  const go = async (id) => {
    // the look of the pet is captured before the trip, so a pet that dies out there
    // still walks into the scene alive
    const look = {parts: pet.parts, acc: pet.acc};
    const next = await act(() => api.hunt(sel, id), true);
    if (next?.hunt) {
      setScene({...next.hunt, pet: look});
    }
  };
  const why = (m) => {
    if (pet.dead) return t('hunt.dead');
    if (pet.sick) return t('hunt.sick');
    if (lv < m.min_lv) return tr(`Cần cấp ${m.min_lv}`, `Level ${m.min_lv} required`, `レベル${m.min_lv}が必要`);
    if (rest) return `⏳ Nghỉ ${rest}`;
    if (pet.hunger < 10) return t('hunt.hungry');
    if (pet.health < 20) return t('hunt.weak');
    return '';
  };
  const selected = (state.maps || []).find((m)=>m.id===pick) || state.maps[0];
  return (
    <div className='vpet-hunt'>
      <nav className='vpet-inv__tabs' aria-label={t('hunt.mode')}><button type='button' aria-pressed={!team} onClick={()=>setTeam(false)}>{t('hunt.solo')}</button><button type='button' aria-pressed={team} onClick={()=>setTeam(true)}>{t('hunt.party')}</button></nav>
      {team ? <Parties state={state} sel={sel} busy={busy} act={act} mapID={pick}/> : <>
        <Realm id={selected.id} className='vpet-hunt__landscape'>
          <div className='vpet-realm__heading'><b>{selected.name}</b><small>{selected.desc}</small></div>
          <div className='vpet-hunt__companion'><PetCanvas pet={pet} parts={pet.parts} scale={3}/></div>
          <div className='vpet-hunt__landmark'><b>{t('hunt.grounds')}</b><small>{t('hunt.level', selected.min_lv)}</small></div>
        </Realm>
        <nav className='vpet-hunt-regions' aria-label={t('hunt.region')}>
          {(state.maps || []).map((m)=><button key={m.id} type='button' className={`vpet-quest-card${lv<m.min_lv?' is-locked':''}${pick===m.id?' is-picked':''}`} aria-pressed={pick===m.id} onClick={()=>setPick(m.id)} style={{backgroundImage:`url(${REALMS[m.id]})`}}><b>{m.name}</b><small>{`Lv ${m.min_lv}+`}</small></button>)}
        </nav>
        <section className='vpet-expedition-bar'>
          <div><b>{selected.name}</b><small>{t('hunt.brief', 10, 10, selected.xp, Math.round(selected.risk * 100))}</small><small>{t('hunt.gold', Math.floor(selected.xp / 2), Math.ceil(selected.xp * 1.5) - 1)}</small></div>
          <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(why(selected))} onClick={()=>go(selected.id)}>{why(selected) || t('hunt.accept')}</button>
        </section>
        <p className='vpet-quest__brief'>{rest?`Thú đang hồi sức, còn ${rest} nữa mới lên đường được.`:'Chọn vùng, kiểm tra độ hiểm nguy rồi xuất phát.'}</p>
      </>}
    </div>
  );
}

// The character is shown once, surrounded by its six equipment slots.
// What a slot's gems add, in the same words the stat sheet uses.
const GEM_STAT = {gem_atk: ['atk', 1.5, '⚔️'], gem_def: ['def', 1.2, '🛡️'], gem_hp: ['hp', 8, '❤️']};
function gemText(gems) {
  const sum = {};
  for (const id of gems) {
    const kind = id.replace(/_\d+$/, '');
    const tier = Number(id.slice(-1));
    const spec = GEM_STAT[kind];
    if (!spec) continue;
    sum[spec[2]] = (sum[spec[2]] || 0) + spec[1] * tier;
  }
  return Object.entries(sum).map(([icon, v]) => `${icon}+${Math.round(v * 10) / 10}`).join(' ');
}

export function GearBox({state, sel, busy, act, children, onInventory}) {
  const [pick, setPick] = useState('');
  const pet = state.pets[sel];
  const slots = state.gear_slots || [];
  const gear = pet.gear || {};
  const levels = pet.gear_level || {};
  const worn = slots.map((s) => gear[s.id]).filter(Boolean).map((id) => itemOf(id)).filter(Boolean);
  // the set bonus is part of what the pet actually fights with, so the doll totals include it
  const set = state.sets?.[sel];
  const sum = (k) => state.gear_totals?.[sel]?.[k] ?? worn.reduce((n, it) => n + ((state.gear_stats || {})[it.id]?.[k] || 0), 0);
  // Every slot is its own wardrobe: an empty slot equips, a filled slot replaces or removes.
  // This keeps the entire wear flow on the character sheet instead of sending players to the bag.
  const fitting = (slotID) => [
    ...(state.catalog || []).filter((c) => state.inv[c.id] > 0 && (state.gear_stats || {})[c.id]?.slot === slotID).map((item) => ({item, level: 0, count: state.inv[item.id]})),
    ...(state.enhanced || []).filter((stack) => (state.gear_stats || {})[stack.item]?.slot === slotID).map((stack) => ({item: itemOf(stack.item), level: stack.level, count: stack.count})).filter((row) => row.item),
  ].sort((a, b) => (b.item.rarity || 0) - (a.item.rarity || 0) || b.level - a.level);
  const frame = (s) => {
    const it = gear[s.id] && itemOf(gear[s.id]);
    const level = levels[s.id] || 0;
    const socketed = (state.sockets?.[sel] || []).find((v) => v.slot === s.id)?.gems || [];
    const open = pick === s.id;
    const choices = open ? fitting(s.id) : [];
    return (
      <div key={s.id} className='vpet-frame-wrap'>
        <button
          type='button'
          className={`${it ? `vpet-frame is-${RARITY[it.rarity || 0]} enhance-${level >= 10 ? 10 : level >= 7 ? 7 : level >= 4 ? 4 : level > 0 ? 1 : 0}` : 'vpet-frame is-empty'}${open ? ' is-open' : ''}`}
          disabled={busy}
          aria-expanded={open}
          aria-haspopup='menu'
          title={it ? tr(`${it.name} — Bấm để đổi hoặc tháo`, `${it.name} — click to swap or remove`, `${it.name} — 交換・外す`) : tr(`${s.name} — Bấm để mặc đồ`, `${s.name} — click to equip`, `${s.name} — 装備する`)}
          onClick={() => setPick(open ? '' : s.id)}
        >
          {it ? (
            <span className='vpet-frame__icon'>
              <ItemIcon id={it.id} size={30}/>
              {level > 0 && <span className='vpet-enhance-badge'>{`+${level}`}</span>}
            </span>
          ) : <span className='vpet-frame__emoji'>{s.emoji}</span>}
          <small>{it ? `${it.name}${level > 0 ? ` +${level}` : ''}` : s.name}</small>
          {it ? <span className='vpet-frame__slot'>{tr(`${s.name} · Đổi`, `${s.name} · Swap`, `${s.name} · 交換`)}</span> : <span className='vpet-frame__equip'>＋ Mặc đồ</span>}
          {it && socketed.length > 0 && (
            <>
              <span className='vpet-frame__gems' aria-hidden='true'>
                {socketed.map((g, i) => <ItemIcon key={`${g}-${i}`} id={g} size={14}/>)}
              </span>
              <span className='vpet-frame__bonus'>{gemText(socketed)}</span>
            </>
          )}
        </button>
        {open && (
          <div className='vpet-frame-pick' role='menu' aria-label={tr(`Trang bị ${s.name}`, `Equip ${s.name}`, `${s.name}を装備`)}>
            <strong>{it ? tr(`Đổi ${s.name}`, `Swap ${s.name}`, `${s.name}を交換`) : tr(`Mặc ${s.name}`, `Equip ${s.name}`, `${s.name}を装備`)}</strong>
            {choices.map((c) => (
              <button key={`${c.item.id}-${c.level}`} type='button' role='menuitem' className={`vpet-frame-pick__item is-${RARITY[c.item.rarity || 0]}`} disabled={busy}
                onClick={() => { setPick(''); act(() => api.equip(sel, c.item.id, c.level)); }}>
                <span className='vpet-frame__icon'>
                  <ItemIcon id={c.item.id} size={20}/>
                  {c.level > 0 && <span className='vpet-enhance-badge'>{`+${c.level}`}</span>}
                </span>
                <span>{`${c.item.name}${c.level > 0 ? ` +${c.level}` : ''}`}</span>
                <small>{`×${c.count}`}</small>
              </button>
            ))}
            {!choices.length && <p>{tr(`Trong túi chưa có ${s.name.toLocaleLowerCase('vi-VN')}.`, `There is no ${s.name.toLowerCase()} in the bag.`, `持ち物に${s.name}がありません。`)}</p>}
            {it && <button type='button' role='menuitem' className='vpet-frame-pick__remove' disabled={busy}
              onClick={() => { setPick(''); act(() => api.unequipSlot(sel, s.id)); }}>{tr(`Tháo ${it.name}`, `Remove ${it.name}`, `${it.name}を外す`)}</button>}
            {onInventory && <button type='button' className='vpet-frame-pick__bag' onClick={() => { setPick(''); onInventory(); }}>{t('pet.openBag')}</button>}
          </div>
        )}
      </div>
    );
  };
  return (
    <div className='vpet-gear'>
      <div className='vpet-doll'>
        <div className='vpet-doll__col'>{slots.slice(0, 4).map(frame)}</div>
        <div className='vpet-doll__mid'>
          {children || <PetCanvas pet={pet} parts={pet.parts} scale={3} still={true}/>}
          <span className='vpet-doll__stats'>
            <span>{`⚔️ +${sum('atk').toFixed(1)}`}<small>{t('hunt.damage')}</small></span>
            <span>{`🛡️ +${sum('def').toFixed(0)}`}<small>{t('hunt.armour')}</small></span>
            <span>{`❤️ +${sum('hp').toFixed(0)}`}<small>{t('hunt.hp')}</small></span>
          </span>
        </div>
        <div className='vpet-doll__col'>{slots.slice(4).map(frame)}</div>
      </div>
      {set ? (
        <p className='vpet-set is-on' role='status'>
          <b>{`✨ ${set.name}`}</b>
          <span>{t('hunt.set', set.parts, slots.length, set.atk.toFixed(1), set.def.toFixed(1), set.hp)}</span>
        </p>
      ) : (
        <p className='vpet-set'>{t('hunt.setHint', SET_MIN)}</p>
      )}
    </div>
  );
}

const SET_MIN = 3;

function skillRankText(sk, rank) {
  const scale = 1 + (rank - 1) * .25;
  const n = Math.round((sk?.base || 0) * scale);
  switch (sk?.kind) {
  case 'multi': return tr(`2–3 phát × ${n} sát thương`, `2–3 hits × ${n} damage`, `2〜3回 × ${n}ダメージ`);
  case 'heal': return tr(`Hồi ${n} máu`, `Restore ${n} HP`, `HPを${n}回復`);
  case 'shield': return tr(`Giảm ${50 + (rank - 1) * 10}% đòn kế tiếp`, `Reduce the next blow by ${50 + (rank - 1) * 10}%`, `次の一撃を${50 + (rank - 1) * 10}%軽減`);
  case 'hide': return rank === 1 ? tr('Né đòn kế tiếp', 'Dodge the next blow', '次の一撃を回避') : tr(`Né đòn kế tiếp · ${20 * (rank - 1)}% tiếp tục ẩn`, `Dodge next blow · ${20 * (rank - 1)}% stay hidden`, `次を回避 · ${20 * (rank - 1)}%で隠れ続ける`);
  case 'stink': return tr(`${n} sát thương · gây hôi`, `${n} damage · inflicts stink`, `${n}ダメージ · 臭い付与`);
  case 'risky': return tr(`${n} sát thương · 50% trúng`, `${n} damage · 50% hit`, `${n}ダメージ · 命中50%`);
  case 'drain': return tr(`${n} sát thương · hồi một nửa`, `${n} damage · heal half`, `${n}ダメージ · 半分回復`);
  case 'damage': return tr(`${n} sát thương`, `${n} damage`, `${n}ダメージ`);
  default: return sk?.desc || '';
  }
}

export function Skills({state, sel, busy, act}) {
  const pet = state.pets[sel];
  const stones = state.inv.ngoc || 0;
  const free = Math.max(0, Math.min(4, 2 + Math.floor((pet.lv || 1) / 10)) - (pet.skills || []).length);
  return (
      <div className='vpet-skills'>
        {(pet.skills || []).map((id) => {
          const sk = (state.skills || []).find((k) => k.id === id);
          const rank = (pet.skill_lv || {})[id] || 1;
          return (
            <div key={id} className='vpet-skill'>
              <b>{`${sk?.emoji || ''} ${sk?.name || id}`}</b>
              <span className='vpet-skill__pips' aria-label={tr(`Cấp ${rank}`, `Rank ${rank}`, `ランク${rank}`)}>
                {[1, 2, 3, 4, 5].map((n) => <i key={n} className={n <= rank ? 'is-on' : ''}/>)}
              </span>
              <small className='vpet-skill__now'>{tr(`Cấp ${rank}: ${skillRankText(sk, rank)}`, `Rank ${rank}: ${skillRankText(sk, rank)}`, `ランク${rank}：${skillRankText(sk, rank)}`)}</small>
              {rank < 5 && <small className='vpet-skill__next'>{tr(`Cấp ${rank + 1}: ${skillRankText(sk, rank + 1)}`, `Rank ${rank + 1}: ${skillRankText(sk, rank + 1)}`, `ランク${rank + 1}：${skillRankText(sk, rank + 1)}`)}</small>}
              {stones > 0 && rank < 5 && (
                <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.skillUp(sel, id))}>{tr(`💠 Nâng cấp (${stones})`, `💠 Upgrade (${stones})`, `💠 強化（${stones}）`)}</button>
              )}
            </div>
          );
        })}
        {free > 0 && <div className='vpet-skill is-free'>{tr(`🔓 Còn ${free} ô kỹ năng trống, lên cấp để học thêm`, `🔓 ${free} skill slots open; level up to learn more`, `🔓 空きスキル枠${free}。レベルアップで習得`)}</div>}
      </div>
  );
}

// Level bar and earned medals.
export function Badges({state, sel}) {
  const pet = state.pets[sel];
  const lv = pet.lv || 1;
  const need = xpNeed(lv);
  const got = (state.medals || [])[sel] || [];
  const title = ((state.titles || [])[sel] || []).find((row) => row.equipped);
  return (
    <div className='vpet-badges'>
      {title && (
        <p className='vpet-worn-title' title={title.desc}>
          <b>{`${title.emoji} ${title.name}`}</b>
          <small>{[title.atk && `⚔️+${Math.round(title.atk * 10) / 10}`, title.def && `🛡️+${Math.round(title.def * 10) / 10}`, title.hp && `❤️+${Math.round(title.hp * 10) / 10}`].filter(Boolean).join(' ')}</small>
        </p>
      )}
      <div className='vpet-lv'>
        <b>{`Lv ${lv}`}</b>
        <div className='vpet-lv__track' role='meter' aria-label={t('hunt.xp')} aria-valuenow={pet.xp || 0} aria-valuemin={0} aria-valuemax={need}>
          <div style={{width: `${Math.min(100, ((pet.xp || 0) / need) * 100)}%`}}/>
        </div>
        <small>{lv >= 50 ? t('hunt.max') : `${pet.xp || 0}/${need} kn`}</small>
      </div>
      {got.length > 0 && (
        <p className='vpet-medals'>
          {got.map((m) => <span key={m.name} className='vpet-medal' title={m.desc}>{m.emoji}<small>{m.name}</small></span>)}
        </p>
      )}
      <p className='vpet-muted'>{tr(`🗺️ ${pet.hunts || 0} chuyến săn · ⚔️ ${pet.wins || 0} thắng / ${pet.losses || 0} thua`, `🗺️ ${pet.hunts || 0} hunts · ⚔️ ${pet.wins || 0} wins / ${pet.losses || 0} losses`, `🗺️ 狩り${pet.hunts || 0}回 · ⚔️ ${pet.wins || 0}勝 / ${pet.losses || 0}敗`)}</p>
    </div>
  );
}
