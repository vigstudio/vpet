import React, {useState} from 'react';

import * as api from './api';
import ItemIcon from './ItemIcon';
import PetCanvas from './PetCanvas';
import {RARITY} from './Hunt';
import {t, tr} from './i18n';
import {ladders, titleGroups, wornTitle} from './quests.cjs';
import {itemOf, select} from './store';
import GameIcon from './GameIcon';
import foodArt from '../assets/fantasy/food-sushi.png';
import trophyArt from '../assets/fantasy/trophy.png';
import coinArt from '../assets/fantasy/coin.png';
import scrollArt from '../assets/fantasy/scroll.png';

// Quests are the first thing a new player can finish, so dailies lead and the longer
// achievements sit underneath rather than competing for the top of the screen.
function Row({q, busy, act, rung, onGo}) {
  const ready = q.have >= q.need && !q.done;
  const destination = {d_feed:'pet', d_hunt:'hunt', d_stage:'world', d_duel:'duel'}[q.id];
  return (
    <li className={`vpet-quest-row${q.done ? ' is-done' : ''}${ready ? ' is-ready' : ''}`}>
      <div className='vpet-quest-row__text'>
        <span className='vpet-quest-row__icon' aria-hidden='true'>{q.id === 'd_feed' ? <img src={foodArt} alt=''/> : destination ? <GameIcon name={destination} size={48}/> : <img src={trophyArt} alt=''/>}</span>
        <b>{q.name}{rung && <i className='vpet-quest-row__rung'>{rung}</i>}</b>
        <small>{q.desc}</small>
      </div>
      <div className='vpet-quest-row__bar' role='meter' aria-valuenow={q.have} aria-valuemin={0} aria-valuemax={q.need} aria-label={`${q.name}: ${q.have}/${q.need}`}>
        <i style={{width: `${Math.min(100, (q.have / q.need) * 100)}%`}}/>
        <span>{`${q.have}/${q.need}`}</span>
      </div>
      <div className='vpet-quest-row__pay'>
        <span><img src={coinArt} width='22' height='22' alt={tr('vàng', 'gold', 'ゴールド')}/>{q.gold}</span>
        {q.item && <span className='vpet-quest-row__item'><ItemIcon id={q.item} size={18}/>{`×${q.count}`}</span>}
      </div>
      <button type='button' className={`vpet-btn${ready ? ' vpet-btn--primary' : ''}`} disabled={busy || q.done || (!ready && !destination)}
        onClick={() => ready ? act(() => api.claimQuest(q.id)) : onGo(destination)}>
        {q.done ? t('quest.claimed') : ready ? t('quest.claim') : destination ? tr('Thực hiện','Go','移動') : t('quest.notDone')}
      </button>
    </li>
  );
}

// Floating point stat steps reach us as 1.7999999999999998; one decimal is all a title bonus
// ever carries, so the label rounds rather than printing the whole mantissa.
const num = (v) => String(Math.round(v * 10) / 10);
const STAT_BITS = (row) => [
  row.atk ? `⚔️+${num(row.atk)}` : '',
  row.def ? `🛡️+${num(row.def)}` : '',
  row.hp ? `❤️+${num(row.hp)}` : '',
].filter(Boolean).join(' ');

// One title is worn at a time, so the board is a shelf of what this pet has earned and a
// single choice among them, not a pile of permanent bonuses.
function Titles({state, sel, busy, act}) {
  const rows = state.titles?.[sel] || [];
  const pet = state.pets?.[sel];
  const groups = titleGroups(rows);
  const worn = wornTitle(rows);
  if (!pet) return <p className='vpet-empty'>{t('inv.hatchFirst')}</p>;
  return (
    <div className='vpet-titles'>
      <header className='vpet-titles__worn'>
        <div>
          <b>{worn ? `${worn.emoji} ${worn.name}` : t('title.none')}</b>
          <small>{worn ? `${pet.name} · ${STAT_BITS(worn)}` : t('title.noneHint', pet.name)}</small>
        </div>
        {worn && <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.wearTitle(sel, ''))}>{t('title.remove')}</button>}
      </header>
      <p className='vpet-note'>{t('title.rule')}</p>
      <div className='vpet-title-groups'>
        {groups.map((g) => (
          <section key={g.track} className='vpet-title-group' aria-label={t(`title.track.${g.track}`)}>
            <header><span aria-hidden='true'>{g.emoji}</span><b>{t(`title.track.${g.track}`)}</b><small>{`${g.unlocked}/${g.rows.length}`}</small></header>
            {g.rows.map((row) => (
              <article key={row.id} className={`vpet-title${row.unlocked ? ' is-open' : ''}${row.equipped ? ' is-worn' : ''}`}>
                <div className='vpet-title__text'>
                  <b>{row.name}</b>
                  <small>{row.desc}</small>
                </div>
                <span className='vpet-title__stats'>{STAT_BITS(row)}</span>
                {row.unlocked ? (
                  <button type='button' className={`vpet-btn${row.equipped ? '' : ' vpet-btn--primary'}`} disabled={busy || row.equipped}
                    onClick={() => act(() => api.wearTitle(sel, row.id))}>
                    {row.equipped ? t('title.worn') : t('title.wear')}
                  </button>
                ) : <span className='vpet-title__lock'>{`${row.have}/${row.need}`}</span>}
              </article>
            ))}
          </section>
        ))}
      </div>
    </div>
  );
}

export default function Quests({state, sel, busy, act, onGo}) {
  const [view, setView] = useState('daily');
  const all = state.quests || [];
  const daily = all.filter((q) => q.daily);
  const rungs = ladders(all);
  const titles = state.titles?.[sel] || [];
  const ready = all.filter((q) => q.have >= q.need && !q.done).length;
  const feats = all.filter((q) => !q.daily);
  const tabs = [
    ['daily', t('quest.dailies', daily.filter((q) => q.done).length, daily.length)],
    ['feats', t('quest.achievements', feats.filter((q) => q.done).length, feats.length)],
    ['titles', t('title.tab', titles.filter((x) => x.unlocked).length, titles.length)],
  ];
  return (
    <div className='vpet-quests'>
      <nav className='vpet-sheet-tabs' role='tablist' aria-label={t('quest.section')}>
        {tabs.map(([id, label]) => (
          <button key={id} type='button' role='tab' aria-selected={view === id} onClick={() => setView(id)}>{label}</button>
        ))}
      </nav>
      {view === 'titles' ? <Titles state={state} sel={sel} busy={busy} act={act}/> : (
        <>
          <p className='vpet-note'>
            {ready > 0 ? tr(`🎯 Có ${ready} nhiệm vụ đã xong, nhận thưởng ngay.`, `🎯 ${ready} quests are ready to claim.`, `🎯 ${ready}件の報酬を受け取れます。`) : t('quest.note')}
          </p>
          <ul className='vpet-quest-list'>
            {view === 'daily'
              ? daily.map((q) => <Row key={q.id} q={q} busy={busy} act={act} onGo={onGo}/>)
              : rungs.map((l) => <Row key={l.quest.id} q={l.quest} busy={busy} act={act} onGo={onGo} rung={l.tiers > 1 ? `${l.cleared}/${l.tiers}` : ''}/>)}
          </ul>
          {view === 'feats' && <p className='vpet-muted'>{t('quest.ladderHint')}</p>}
        </>
      )}
      <aside className='vpet-quests__guide'>
        <img src={scrollArt} width='100' height='100' alt=''/>
        <h3>{t('quest.section')}</h3>
        <p>{t('quest.note')}</p>
        <b>{tr(`${ready} nhiệm vụ có thể nhận`, `${ready} quests ready to claim`, `${ready}件の報酬を受取可能`)}</b>
        <p>{t('quest.ladderHint')}</p>
      </aside>
    </div>
  );
}

// Crushing and socketing share a screen: junk goes in on the left, gems come out and go
// straight into the gear shown on the right.
export function Gems({state, sel, busy, act}) {
  const inv = state.inv || {};
  // Mirrors crushable() on the server: anything below the forge-only rungs can be crushed.
  const junk = (state.catalog || []).filter((it) => inv[it.id] > 0 && (it.id === 'da' || (state.gear_stats?.[it.id] && (it.rarity || 0) < 4)));
  // Mirrors shardValue(): each rung is worth three of the one below it.
  const shardsFor = (it) => (it.id === 'da' ? 1 : 3 ** (it.rarity || 0));
  const shards = inv.gem_shard || 0;
  const gems = (state.catalog || []).filter((it) => it.kind === 'gem' && it.id !== 'gem_shard' && inv[it.id] > 0);
  const sockets = state.sockets?.[sel] || [];
  const pet = state.pets?.[sel];
  const alive = (state.pets || []).map((p, i) => [p, i]).filter(([p]) => !p.dead);
  const [picking, setPicking] = useState('');
  const kinds = [['gem_atk', t('gem.ruby'), t('gem.damage')], ['gem_def', t('gem.sapphire'), t('gem.armour')], ['gem_hp', t('gem.emerald'), t('gem.hp')]];
  return (
    <div className='vpet-gems'>
      <p className='vpet-note'>{t('gem.shards', shards)}</p>
      <div className='vpet-gems__body'>
        <section className='vpet-gems__crush' aria-label={t('gem.crush')}>
          <h4>{t('gem.crush')}</h4>
          {junk.length ? (
            <ul className='vpet-gems__junk'>
              {junk.map((it) => {
                const value = shardsFor(it);
                const count = inv[it.id];
                const crushAll = () => {
                  if (window.confirm(t('gem.confirmAll', count, it.name, count * value))) {
                    act(() => api.crushGem(it.id, count));
                  }
                };
                return (
                  <li key={it.id} className={`is-${RARITY[it.rarity || 0]}`}>
                    <ItemIcon id={it.id} size={28}/>
                    <span>{it.name}<small>{`${state.rarities?.[it.rarity || 0] || ''} · ×${count} · ${t('gem.value', value)}`}</small></span>
                    <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.crushGem(it.id, 1))}>{t('gem.crush1')}</button>
                    {count > 1 && <button type='button' className='vpet-btn' disabled={busy} onClick={crushAll}>{t('gem.crushAll', count * value)}</button>}
                  </li>
                );
              })}
            </ul>
          ) : <p className='vpet-muted'>{t('gem.none')}</p>}
          <h4>{t('gem.forge')}</h4>
          <div className='vpet-gems__forge'>
            {kinds.map(([id, name, stat]) => (
              <article key={id}>
                <b>{name}</b>
                <small>{tr(`Cộng ${stat}`, `Adds ${stat}`, `${stat}を追加`)}</small>
                <button type='button' className='vpet-btn' disabled={busy || shards < 6} onClick={() => act(() => api.forgeGem(id, 1))}>{t('gem.first')}</button>
                {[2, 3, 4].map((tier) => (
                  <button key={tier} type='button' className='vpet-btn' disabled={busy || (inv[`${id}_${tier - 1}`] || 0) < 3}
                    onClick={() => act(() => api.forgeGem(id, tier))}>{t('gem.next', tier)}</button>
                ))}
              </article>
            ))}
          </div>
        </section>
        <aside className='vpet-gems__socket' aria-label={t('gem.socket')}>
          <h4>{t('gem.socketDesc')}</h4>
          {/* Gems are seated on one pet's gear, so the panel has to say which pet that is. */}
          <div className='vpet-gems__who'>
            {alive.length > 1 && (
              <div className='vpet-switch' role='tablist' aria-label={t('gem.forPet')}>
                {alive.map(([p, i]) => (
                  <button key={p.born} type='button' role='tab' aria-selected={i === sel} className={i === sel ? 'is-active' : ''} onClick={() => select(i)}>
                    <PetCanvas pet={p} parts={p.parts} scale={1} still={true}/>
                    {p.name}
                  </button>
                ))}
              </div>
            )}
            {pet && (
              <div className='vpet-gems__pet'>
                <PetCanvas pet={pet} parts={pet.parts} scale={2} still={true}/>
                <div>
                  <b>{pet.name}</b>
                  <small>{t('gem.wearing', sockets.length)}</small>
                </div>
              </div>
            )}
          </div>
          {sockets.length ? (
            <div className='vpet-gems__gear'>
              {sockets.map((s) => {
                const item = itemOf(s.item);
                const open = picking === s.slot;
                return (
                  <article key={s.slot} className={`vpet-socket-row is-${RARITY[item?.rarity || 0]}`}>
                    <header>
                      <ItemIcon id={s.item} size={24}/>
                      <b>{item?.name || s.item}</b>
                      <small>{`${s.gems.length}/${s.gems.length + s.free}`}</small>
                    </header>
                    <div className='vpet-socket-row__slots'>
                      {s.gems.map((g, i) => (
                        <button key={`${g}-${i}`} type='button' className='vpet-socket is-filled' title={t('gem.removeTitle', itemOf(g)?.name || g, 25)}
                          disabled={busy} onClick={() => act(() => api.unsocketGem(sel, s.slot, i))}>
                          <ItemIcon id={g} size={20}/>
                        </button>
                      ))}
                      {Array.from({length: s.free}, (_, i) => (
                        <button key={`free-${i}`} type='button' className='vpet-socket' disabled={busy || !gems.length}
                          aria-expanded={open} title={gems.length ? t('gem.addTitle') : t('gem.noGems')}
                          onClick={() => setPicking(open ? '' : s.slot)}
                        >{'+'}</button>
                      ))}
                    </div>
                    {open && gems.length > 0 && (
                      <div className='vpet-socket-row__pick'>
                        {gems.map((g) => (
                          <button key={g.id} type='button' className='vpet-btn' disabled={busy}
                            onClick={() => { setPicking(''); act(() => api.socketGem(sel, s.slot, g.id)); }}>
                            <ItemIcon id={g.id} size={16}/>{`${g.name} (${inv[g.id]})`}
                          </button>
                        ))}
                      </div>
                    )}
                  </article>
                );
              })}
            </div>
          ) : <p className='vpet-muted'>{t('gem.noGear')}</p>}
        </aside>
      </div>
    </div>
  );
}
