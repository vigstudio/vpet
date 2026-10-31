import React, {useState} from 'react';

import * as api from './api';
import {benchActions, benchItems} from './forge.cjs';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import {t, tr} from './i18n';
import {clearNewItem, itemOf, select, useVPet} from './store';
import {Gems} from './Quests';

const CATEGORIES = [
  ['all', 'inv.all', () => true],
  ['gear', 'inv.gear', (it) => it.kind === 'gear' || it.kind === 'stone'],
  ['use', 'inv.consume', (it) => ['food', 'med', 'revive', 'energy'].includes(it.kind)],
  ['gem', 'inv.gem', (it) => it.kind === 'gem'],
  ['misc', 'inv.misc', (it) => ['bag', 'acc', 'card', 'junk'].includes(it.kind)],
];
const KINDS = {gem: 'kind.gem', energy: 'kind.energy', gear: 'kind.gear', stone: 'kind.stone', food: 'kind.food', med: 'kind.med', revive: 'kind.revive', acc: 'kind.acc', bag: 'kind.bag', card: 'kind.card', junk: 'kind.junk'};
const STATS = [['atk', 'pet.damage'], ['def', 'pet.armour'], ['hp', 'pet.battleHP']];
const VERBS = {energy: 'inv.drink', gear: 'inv.equipAction', food: 'inv.feed', med: 'pet.use', revive: 'kind.revive', acc: 'inv.wear', bag: 'inv.open'};

function unavailable(it, pet, state) {
  if (!it || !state.inv[it.id]) return t('inv.notOwned');
  if (it.kind === 'junk') return t('inv.noJunkRecipe');
  if (it.kind === 'stone') return t('inv.stoneHelp');
  if (it.kind === 'bag') return '';
  if (it.id === 'the_slot') return state.slots >= 4 ? t('inv.fullStable') : '';
  if (!pet) return t('inv.hatchFirst');
  if (it.kind === 'revive') return pet.dead ? '' : t('inv.reviveOnly');
  if (pet.dead) return t('inv.deadFirst');
  if (it.kind === 'food' && pet.hunger >= 95) return t('inv.fullPet');
  return '';
}

const glow = (level) => `enhance-${level >= 10 ? 10 : level >= 7 ? 7 : level >= 4 ? 4 : level > 0 ? 1 : 0}`;

// One bench, one piece of gear on it, and every mechanic that can touch that piece listed
// underneath. Grouping by mechanic instead - fuse here, enhance there, sockets on a third
// screen - is what made this unreadable: nobody sets out to "socket", they set out to make
// one sword better.
function ForgeBench({state, sel, busy, act, onReceipt}) {
  const [pick, setPick] = useState('');
  const [confirm, setConfirm] = useState('');
  const rows = benchItems({
    worn: state.enhance?.[sel] || [],
    inv: state.inv || {},
    enhanced: state.enhanced || [],
    gearStats: state.gear_stats || {},
    catalog: state.catalog || [],
  });
  const row = rows.find((r) => r.key === pick) || rows[0];
  const shards = state.inv?.gem_shard || 0;
  const actions = benchActions(row, {
    enhance: state.enhance?.[sel] || [],
    sockets: state.sockets?.[sel] || [],
    recipes: state.forge || [],
    coin: state.coin || 0,
    shards,
    catalog: state.catalog || [],
  });
  const gems = (state.catalog || []).filter((it) => it.kind === 'gem' && it.id !== 'gem_shard' && state.inv[it.id] > 0);
  const slotName = (id) => (state.gear_slots || []).find((s) => s.id === id)?.name || id;
  const run = async (call) => {
    const next = await act(call, true);
    setConfirm('');
    if (next) onReceipt(next.message);
  };
  if (!rows.length) return <p className='vpet-inv__empty'>{t('forge.noGear')}</p>;

  const find = (kind) => actions.find((a) => a.kind === kind);
  const enhance = find('enhance');
  const socket = find('socket');
  const fuse = find('fuse');
  const restore = find('restore');

  return (
    <div className='vpet-bench'>
      <div className='vpet-bench__list' role='group' aria-label={t('forge.pick')}>
        {rows.map((r) => (
          <button key={r.key} type='button' aria-pressed={r.key === row.key}
            className={`vpet-slot is-${RARITY[r.item.rarity || 0]}${r.key === row.key ? ' is-picked' : ''} ${glow(r.level)}`}
            title={`${r.item.name}${r.level ? ` +${r.level}` : ''}`}
            onClick={() => { setPick(r.key); setConfirm(''); }}>
            <span className='vpet-frame__icon'>
              <ItemIcon id={r.item.id} size={32}/>
              {r.level > 0 && <span className='vpet-enhance-badge'>{`+${r.level}`}</span>}
            </span>
            <span className='vpet-slot__name'>{r.item.name}</span>
            <span className='vpet-slot__n'>{r.worn ? t('forge.wornTag') : `×${r.count}`}</span>
          </button>
        ))}
      </div>
      <aside className='vpet-bench__panel' aria-label={t('forge.bench')}>
        <header className={`vpet-bench__head is-${RARITY[row.item.rarity || 0]}`}>
          <span className='vpet-frame__icon'>
            <ItemIcon id={row.item.id} size={44}/>
            {row.level > 0 && <span className='vpet-enhance-badge'>{`+${row.level}`}</span>}
          </span>
          <div>
            <b>{`${row.item.name}${row.level > 0 ? ` +${row.level}` : ''}`}</b>
            <small>{`${state.rarities?.[row.item.rarity || 0] || ''} · ${slotName(row.slot)}${row.worn ? ` · ${t('forge.wornTag')}` : ''}`}</small>
          </div>
          <span className='vpet-bench__purse'>{t('forge.purse', state.coin || 0, shards)}</span>
        </header>

        {/* 1. enhance */}
        <section className='vpet-bench__act'>
          <header><b>{t('forge.enhance')}</b>{enhance?.max && <small>{`+${enhance.level} / +${enhance.max}`}</small>}</header>
          {enhance?.blocked === 'equip' ? (
            <p className='vpet-bench__why'>{t('forge.needEquip')}
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => run(() => api.equip(sel, row.item.id, row.level))}>{t('forge.equipNow')}</button>
            </p>
          ) : enhance?.blocked === 'max' ? (
            <p className='vpet-bench__why'>{t('forge.atMax', enhance.max)}</p>
          ) : enhance ? (
            <>
              <dl className='vpet-bench__stats'>
                {STATS.map(([key, name]) => (
                  <div key={key}><dt>{t(name)}</dt><dd>{enhance.current[key] || 0}<strong>{` → ${enhance.next[key] || 0}`}</strong></dd></div>
                ))}
              </dl>
              <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(enhance.blocked)}
                onClick={() => run(() => api.enhance(sel, enhance.slot))}>
                {t('forge.enhanceTo', enhance.level + 1, enhance.coin, enhance.shards)}
              </button>
              {enhance.blocked === 'cost' && <p className='vpet-bench__why'>{t('forge.short', enhance.shortCoin, enhance.shortShards)}</p>}
            </>
          ) : null}
        </section>

        {/* 2. sockets */}
        <section className='vpet-bench__act'>
          <header><b>{t('forge.socket')}</b>{socket?.gems && <small>{`${socket.gems.length}/${socket.gems.length + socket.free}`}</small>}</header>
          {socket?.blocked === 'equip' ? <p className='vpet-bench__why'>{t('forge.needEquipGem')}</p> : (
            <>
              <div className='vpet-socket-row__slots'>
                {socket.gems.map((g, i) => (
                  <button key={`${g}-${i}`} type='button' className='vpet-socket is-filled' disabled={busy}
                    title={t('gem.removeTitle', itemOf(g)?.name || g, 25)}
                    onClick={() => run(() => api.unsocketGem(sel, socket.slot, i))}><ItemIcon id={g} size={20}/></button>
                ))}
                {Array.from({length: socket.free}, (_, i) => (
                  <button key={`free-${i}`} type='button' className='vpet-socket' disabled={busy || !gems.length}
                    aria-expanded={confirm === 'gem'} title={gems.length ? t('gem.addTitle') : t('gem.noGems')}
                    onClick={() => setConfirm(confirm === 'gem' ? '' : 'gem')}>{'+'}</button>
                ))}
              </div>
              {confirm === 'gem' && gems.length > 0 && (
                <div className='vpet-socket-row__pick'>
                  {gems.map((g) => (
                    <button key={g.id} type='button' className='vpet-btn' disabled={busy}
                      onClick={() => run(() => api.socketGem(sel, socket.slot, g.id))}>
                      <ItemIcon id={g.id} size={16}/>{`${g.name} (${state.inv[g.id]})`}
                    </button>
                  ))}
                </div>
              )}
              {!gems.length && <p className='vpet-bench__why'>{t('forge.noGems')}</p>}
            </>
          )}
        </section>

        {/* 3. fuse up a tier */}
        <section className='vpet-bench__act'>
          <header><b>{t('forge.fuse')}</b>{fuse?.recipe && <small>{`${Math.min(fuse.have, fuse.need)}/${fuse.need}`}</small>}</header>
          {fuse?.blocked === 'top' ? <p className='vpet-bench__why'>{t('inv.maxNoRecipe')}</p>
            : fuse?.blocked === 'enhanced' ? <p className='vpet-bench__why'>{t('forge.restoreFirst', fuse.level)}</p>
              : fuse ? (
                <>
                  <div className='vpet-bench__into'>
                    <ItemIcon id={fuse.into?.id} size={32}/>
                    <div><small>{t('inv.receive')}</small><b>{fuse.into?.name}</b></div>
                  </div>
                  {fuse.recipe?.material && <p className='vpet-bench__why'>{itemOf(fuse.recipe.material)?.name}: {fuse.recipe.material_have}/{fuse.recipe.material_need}</p>}
                  {confirm === 'fuse' ? (
                    <div className='vpet-fusion__confirm'>
                      <p>{t('forge.fuseConfirm', fuse.need, row.item.name, fuse.cost, fuse.into?.name)}</p>
                      <button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm('')}>{t('app.cancel')}</button>
                      <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy} onClick={() => run(() => api.fuse(row.item.id))}>{t('inv.confirmForge')}</button>
                    </div>
                  ) : (
                    <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(fuse.blocked)} onClick={() => setConfirm('fuse')}>
                      {t('forge.fuseCost', fuse.need, fuse.cost)}
                    </button>
                  )}
                  {fuse.blocked === 'parts' && <p className='vpet-bench__why'>{t('forge.shortParts', fuse.shortParts)}</p>}
                  {fuse.blocked === 'cost' && <p className='vpet-bench__why'>{t('forge.shortCoin', fuse.shortCoin)}</p>}
                </>
              ) : null}
        </section>

        {restore && (
          <section className='vpet-bench__act'>
            <header><b>{t('forge.restore')}</b></header>
            {confirm === 'restore' ? (
              <div className='vpet-fusion__confirm'>
                <p>{t('forge.restoreConfirm', restore.level)}</p>
                <button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm('')}>{t('app.cancel')}</button>
                <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy} onClick={() => run(() => api.restoreEnhanced(restore.item, restore.level))}>{tr('Hoàn nguyên', 'Restore +0', '+0に戻す')}</button>
              </div>
            ) : (
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => setConfirm('restore')}>{t('forge.restoreTo')}</button>
            )}
          </section>
        )}
      </aside>
    </div>
  );
}

function SkillSwap({state, sel, busy, act, onReceipt}) {
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const pet = state.pets?.[sel];
  const known = pet?.skills || [];
  const skills = state.skills || [];
  const valid = known.includes(from) && !known.includes(to) && skills.some((skill) => skill.id === to);
  const swap = async () => {
    if (!valid || busy) return;
    const next = await act(() => api.swapSkill(sel, from, to), true);
    if (next) { onReceipt(next.message); setFrom(''); setTo(''); }
  };
  return <div className='vpet-skill-swap'>
    <p>{tr('Chỉ đổi kỹ năng bạn chọn; giữ nguyên cấp và các kỹ năng khác. Tốn 1 giấy.', 'Replace only the chosen skill; keep its level and every other skill. Costs 1 manual.', '選択したスキルだけ変更。レベルと他のスキルは維持。変更書1枚消費。')}</p>
    <fieldset><legend>{tr('Kỹ năng cần thay', 'Skill to replace', '変更するスキル')}</legend>{skills.filter((skill) => known.includes(skill.id)).map((skill) => <button key={skill.id} type='button' className='vpet-btn' aria-pressed={from === skill.id} onClick={() => setFrom(skill.id)}>{skill.emoji} {skill.name}</button>)}</fieldset>
    <fieldset><legend>{tr('Kỹ năng muốn nhận', 'New skill', '新しいスキル')}</legend>{skills.filter((skill) => !known.includes(skill.id)).map((skill) => <button key={skill.id} type='button' className='vpet-btn' aria-pressed={to === skill.id} title={skill.desc} onClick={() => setTo(skill.id)}>{skill.emoji} {skill.name}</button>)}</fieldset>
    <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !valid || !pet || pet.dead || !state.inv.sach} onClick={swap}>{tr('Xác nhận đổi kỹ năng', 'Confirm skill replacement', 'スキル変更を確定')}</button>
  </div>;
}

export default function Inventory({state, sel, busy, act, onReskin, onSkills, onSell}) {
  const [mode, setMode] = useState('items');
  const [cat, setCat] = useState('all');
  const [query, setQuery] = useState('');
  const [pick, setPick] = useState('');
  const [confirm, setConfirm] = useState('');
  const [receipt, setReceipt] = useState('');
  const [quickConfirm, setQuickConfirm] = useState(false);
  const {newItems} = useVPet();
  const pet = state.pets[sel];
  const forge = mode === 'forge';
  const owned = state.catalog.filter((it) => state.inv[it.id] > 0);
  const recipes = state.forge || [];
  const filter = CATEGORIES.find(([id]) => id === cat)[2];
  const items = owned.filter((it) => (forge ? it.kind === 'gear' : filter(it)) && it.name.toLocaleLowerCase().includes(query.toLocaleLowerCase()));
  const item = items.find((it) => it.id === pick) || items[0];
  const recipe = item && recipes.find((r) => r.from === item.id);
  const output = recipe && state.catalog.find((it) => it.id === recipe.to);
  const stats = (it) => (it && state.gear_stats?.[it.id]) || {};
  const wornID = item?.kind === 'gear' && pet?.gear?.[stats(item).slot];
  const worn = state.catalog.find((it) => it.id === wornID);
  const slotName = (state.gear_slots || []).find((s) => s.id === stats(item).slot)?.name;
  const reason = wornID === item?.id ? t('inv.sameGear') : unavailable(item, pet, state);
  const choose = (id) => { setPick(id); setConfirm(''); clearNewItem(id); };
  const changeMode = (value) => { setMode(value); setPick(''); setConfirm(''); setQuickConfirm(false); setQuery(''); setReceipt(''); };

  const use = async () => {
    if (busy || reason) return;
    if (item.id === 'the_ten') { onSkills(); return; }
    if (item.id === 'the_skin') {
      onReskin();
      return;
    }
    const next = await act(() => item.kind === 'gear' ? api.equip(sel, item.id) : api.useItem(sel, item.id), true);
    if (next) setReceipt(next.message || t('inv.used'));
  };
  const fuse = async () => {
    if (busy || !recipe?.ready || confirm !== item.id) return;
    const to = recipe.to;
    const next = await act(() => api.fuse(item.id), true);
    setConfirm('');
    if (next) {
      setReceipt(next.message);
      setMode('items');
      setCat('gear');
      setQuery('');
      setPick(to);
    }
  };
  const fuseAll = async () => {
    if (busy || !recipes.some((r) => r.ready) || !quickConfirm) return;
    const next = await act(() => api.fuseAll(), true);
    setQuickConfirm(false);
    if (next) setReceipt(next.message);
  };
  const equipBest = async () => {
    if (busy || !pet || pet.dead) return;
    const next = await act(() => api.equipBest(sel), true);
    if (next) setReceipt(next.message);
  };

  if (mode === 'gem') {
    return (
      <div className='vpet-inv'>
        <div className='vpet-inv__toolbar'>
          <div className='vpet-inv__modes' role='group' aria-label={t('inv.mode')}>
            <button type='button' aria-pressed={false} onClick={() => changeMode('items')}>{t('inv.items')}</button>
            <button type='button' aria-pressed={false} onClick={() => changeMode('forge')}>{t('inv.forge')}</button>
            <button type='button' aria-pressed={true}>{t('inv.gems')}</button>
          </div>
        </div>
        <Gems state={state} sel={sel} busy={busy} act={act}/>
      </div>
    );
  }
  return (
    <div className='vpet-inv'>
      <div className='vpet-inv__toolbar'>
        <div className='vpet-inv__modes' role='group' aria-label={t('inv.mode')}>
          <button type='button' aria-pressed={!forge} onClick={() => changeMode('items')}>{t('inv.items')}</button>
          <button type='button' aria-pressed={forge} onClick={() => changeMode('forge')}>{t('inv.forge')}</button>
          <button type='button' aria-pressed={mode === 'gem'} onClick={() => changeMode('gem')}>{t('inv.gems')}</button>
        </div>
        <div className='vpet-inv__pet-tools'>
          <label className='vpet-inv__pet'>
            {t('inv.forPet')}
            <select aria-label={t('inv.petReceiver')} value={pet ? sel : ''} disabled={busy || !state.pets.length} onChange={(e) => { select(Number(e.target.value)); setConfirm(''); }}>
              {!state.pets.length && <option value=''>{t('inv.noPet')}</option>}
              {state.pets.map((p, i) => <option key={p.born} value={i}>{`${p.name} · Lv ${p.lv || 1}${p.dead ? t('inv.deadSuffix') : ''}`}</option>)}
            </select>
          </label>
          <button type='button' className='vpet-btn vpet-inv__auto' disabled={busy || !pet || pet.dead} onClick={equipBest}>
            {busy ? t('inv.processing') : t('inv.autoEquip')}
          </button>
        </div>
      </div>
      {receipt && <div className='vpet-inv__receipt' role='status'>{receipt}</div>}
      {forge ? (
        <>
          <div className='vpet-inv__quick-forge'>
            <p className='vpet-inv__help'>{t('forge.help')}</p>
            {quickConfirm ? (
              <div className='vpet-inv__quick-confirm' role='group' aria-label={t('inv.quickForge')}>
                <span>{t('inv.quickForgeConfirm')}</span>
                <button type='button' className='vpet-btn' disabled={busy} onClick={() => setQuickConfirm(false)}>{t('app.cancel')}</button>
                <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy} onClick={fuseAll}>{busy ? t('inv.forging') : t('inv.quickForge')}</button>
              </div>
            ) : (
              <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !recipes.some((r) => r.ready)} onClick={() => setQuickConfirm(true)}>{t('inv.quickForge')}</button>
            )}
          </div>
          <ForgeBench state={state} sel={sel} busy={busy} act={act} onReceipt={setReceipt}/>
        </>
      ) : (
        <>
        <nav className='vpet-inv__tabs' aria-label={t('inv.type')}>
          {CATEGORIES.map(([id, name, f]) => (
            <button key={id} type='button' aria-pressed={cat === id} className={cat === id ? 'is-active' : ''} onClick={() => { setCat(id); choose(''); }}>
              {t(name)}<small>{owned.filter(f).length}</small>
            </button>
          ))}
        </nav>
      <div className='vpet-inv__body'>
        <div className='vpet-inv__list'>
          <label className='vpet-inv__search-label'>
            {t('inv.search')}
            <input className='vpet-inv__search' type='search' placeholder={t('inv.searchHint')} value={query} onChange={(e) => { setQuery(e.target.value); setConfirm(''); }}/>
          </label>
          <div className='vpet-inv__grid' role='group' aria-label={t('inv.pickItem')}>
            {items.map((it) => (
              <button key={it.id} type='button' title={it.name} aria-label={tr(`${it.name}, có ${state.inv[it.id]}`, `${it.name}, have ${state.inv[it.id]}`, `${it.name}、所持${state.inv[it.id]}`)} aria-pressed={item?.id === it.id}
                className={`vpet-slot is-${RARITY[it.rarity || 0]}${item?.id === it.id ? ' is-picked' : ''}`} onClick={() => choose(it.id)}>
                <ItemIcon id={it.id} size={32}/>
                <span className='vpet-slot__name'>{it.name}</span>
                <span className='vpet-slot__n'>{`×${state.inv[it.id]}`}</span>
                {newItems.includes(it.id) && <span className='vpet-slot__new'>{t('inv.new')}</span>}
              </button>
            ))}
            {!items.length && <p className='vpet-inv__empty'>{query ? t('inv.noSearch') : t('inv.empty')}</p>}
          </div>
          <small className='vpet-muted'>{t('inv.count', items.length)}</small>
        </div>
        <aside className='vpet-inv__detail' aria-label={t('inv.detail')}>
          {item ? (
            <>
              <div className={`vpet-inv__head is-${RARITY[item.rarity || 0]}`}>
                <ItemIcon id={item.id} size={48}/>
                <div><b>{item.name}</b><small>{`${state.rarities?.[item.rarity || 0] || ''} · ${slotName || t(KINDS[item.kind]) || ''}`}</small></div>
              </div>
              <div className='vpet-inv__info'>
                <p className='vpet-inv__desc'>{item.desc}</p>
                {item.kind === 'gear' ? (
                  <div className='vpet-inv__comparison'>
                    <p>{worn ? tr(`Đang đeo: ${worn.name}`, `Wearing: ${worn.name}`, `装備中：${worn.name}`) : tr(`${slotName || t('inv.slot')} đang trống`, `${slotName || t('inv.slot')} is empty`, `${slotName || t('inv.slot')}は空き`)}</p>
                    <table className='vpet-inv__stats'>
                      <thead><tr><th scope='col'>{t('inv.stat')}</th><th scope='col'>{t('inv.worn')}</th><th scope='col'>{t('inv.selected')}</th><th scope='col'>{t('inv.change')}</th></tr></thead>
                      <tbody>{STATS.map(([key, nameKey]) => {
                        const before = stats(worn)[key] || 0;
                        const after = stats(item)[key] || 0;
                        const diff = after - before;
                        return <tr key={key}><th scope='row'>{t(nameKey)}</th><td>{before}</td><td>{after}</td><td className={diff < 0 ? 'is-bad' : diff > 0 ? 'is-good' : ''}>{diff > 0 ? `+${diff}` : diff}</td></tr>;
                      })}</tbody>
                    </table>
                    {worn && <small>{t('inv.oldReturns')}</small>}
                  </div>
                ) : (
                  <p className='vpet-muted'>{pet ? `${pet.name}: sức khỏe ${Math.round(pet.health)}/100, độ no ${Math.round(pet.hunger)}/100` : t('inv.noPetHelp')}</p>
                )}
              </div>
              <div className='vpet-inv__footer'>
                {reason && <p className='vpet-inv__warning'>{reason}</p>}
                {item.id === 'sach' ? <SkillSwap key={sel} state={state} sel={sel} busy={busy} act={act} onReceipt={setReceipt}/> : item.kind === 'stone' ? <button type='button' className='vpet-btn vpet-btn--primary vpet-inv__use' onClick={onSkills}>{t('inv.skillUpgrade')}</button> : (
                  <button type='button' className='vpet-btn vpet-btn--primary vpet-inv__use' disabled={busy || Boolean(reason)} onClick={use}>{busy ? t('inv.processing') : `${t(VERBS[item.kind] || 'pet.use')}${pet && !['bag', 'card'].includes(item.kind) ? ` cho ${pet.name}` : ''}`}</button>
                )}
                {recipe && <button type='button' className='vpet-btn vpet-inv__forge-link' onClick={() => { setMode('forge'); setConfirm(''); }}>{tr(`Ghép lên ${output.name}`, `Forge into ${output.name}`, `${output.name}へ鍛造`)}</button>}
                {onSell && item.kind !== 'junk' && <button type='button' className='vpet-btn vpet-inv__sell' onClick={() => onSell(item.id)}>{t('inv.sell')}</button>}
              </div>
            </>
          ) : <p className='vpet-empty'>{t('inv.pickLeft')}</p>}
        </aside>
      </div>
      </>
      )}
    </div>
  );
}
