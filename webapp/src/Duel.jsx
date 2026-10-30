import React, {useEffect, useState} from 'react';

import * as api from './api';
import {dateLocale, t, tr} from './i18n';
import tiles from '../assets/world-tiles.png';
import PetCanvas from './PetCanvas';
import {monsterStyle} from './World';
import UserPicker from './UserPicker';

export const skillText = (state, ids) => (ids || []).map((id) => {
  const s = state.skills?.find((k) => k.id === id);
  return s ? `${s.emoji} ${s.name}` : id;
}).join(' · ');

// Text box that types itself out, like a handheld RPG.
function Typed({text}) {
  const [n, setN] = useState(0);
  useEffect(() => {
    setN(0);
    const t = setInterval(() => setN((x) => (x >= text.length ? (clearInterval(t), x) : x + 1)), 18);
    return () => clearInterval(t);
  }, [text]);
  return <>{text.slice(0, n)}</>;
}

// skills that act on the user instead of flying at the foe
const SELF = new Set(['hoi', 'khien', 'an']);
const SHOT_COLOR = {lua: '#ff7a1a', cuoc: '#ffd23f', bom: '#8bc34a', set: '#ffe94a', hut: '#c2185b', '': '#ffffff'};

const hpTone = (v, max) => (v / max > 0.5 ? 'is-ok' : v / max > 0.2 ? 'is-mid' : 'is-low');

// Replays a finished duel turn by turn on a perspective field: your pet close up at the
// bottom left, the opponent further back at the top right. Hits throw the skill across,
// the target flashes and loses HP; heals and shields glow on the user.
export function Replay({duel, state, onClose}) {
  const [n, setN] = useState(0);
  const [paused, setPaused] = useState(false);
  const [speed, setSpeed] = useState(1);
  const mine = (state.pets || []).some((p) => p.born === duel.b.born) && !(state.pets || []).some((p) => p.born === duel.a.born) ? 1 : 0;
  const foe = 1 - mine;
  useEffect(() => {
    if (paused || n >= duel.turns.length) {
      return undefined;
    }
    const t = setTimeout(() => setN((x) => x + 1), (n ? 1900 : 1600) / speed);
    return () => clearTimeout(t);
  }, [n, duel, paused, speed]);
  const sides = [duel.a, duel.b];
  const turn = n > 0 ? duel.turns[n - 1] : null;
  const before = n > 1 ? duel.turns[n - 2].hp : [duel.a.hp, duel.b.hp];
  const hp = turn ? turn.hp : before;
  const done = n >= duel.turns.length;
  const sudden = (duel.turns || []).some((x) => x.sudden);
  const target = turn ? 1 - turn.by : -1;
  const dmg = turn ? Math.round(before[target] - hp[target]) : 0;
  const crit = Boolean(turn?.text?.includes(t('duel.critical')));
  const skill = turn?.skill || '';
  const fx = skill ? state.skills?.find((k) => k.id === skill)?.emoji : '💥';
  const live = Boolean(turn) && !done;
  const attack = live && !SELF.has(skill);
  const log = done ? (sudden ? t('duel.suddenWinner', sides[duel.winner].name) : t('duel.winner', sides[duel.winner].name)) : (turn?.text || t('duel.intro', sides[foe].name, sides[mine].name));

  const info = (i, where) => (
    <div className={`vpet-arena__info is-${where}`}>
      <b>{sides[i].name}<small>{`Lv ${sides[i].lv || 1}`}</small></b>
      <div className='vpet-arena__hp' role='meter' aria-label={`Máu ${sides[i].name}`} aria-valuenow={Math.round(hp[i])} aria-valuemin={0} aria-valuemax={sides[i].hp}>
        <span>{'HP'}</span>
        <div><i className={hpTone(hp[i], sides[i].hp)} style={{width: `${Math.max(hp[i], 0) / sides[i].hp * 100}%`}}/></div>
      </div>
      {<small>{`${Math.max(Math.round(hp[i]), 0)} / ${Math.round(sides[i].hp)}`}</small>}
    </div>
  );
  // each turn plays in order: attacker lunges, the move flies across, then the impact
  const fighter = (i, where) => {
    const cls = ['vpet-arena__fighter', `is-${where}`];
    if (n === 0) cls.push('is-intro');
    if (live && turn.by === i) cls.push(attack ? 'is-attacking' : 'is-casting');
    if (attack && target === i && dmg > 0) cls.push('is-hit');
    if (done && duel.winner !== i) cls.push('is-fainted');
    return (
      <div className={cls.join(' ')}>
        <div className='vpet-arena__pad'/>
        <div key={live && (turn.by === i || target === i) ? `t${n}` : 'idle'} className='vpet-arena__body'>
          {sides[i].sprite ? (
            <span className={`vpet-arena__monster is-${where}`} style={monsterStyle(sides[i].sprite - 1)} aria-hidden='true'/>
          ) : (
            <PetCanvas pet={{gear: sides[i].gear, acc: sides[i].acc, effect: done && duel.winner === i ? 'cuoi' : ''}} parts={sides[i].parts} scale={where === 'near' ? 4 : 3}/>
          )}
        </div>
        {attack && target === i && <span key={`d${n}`} className={dmg > 0 ? (crit ? 'vpet-arena__dmg is-crit' : 'vpet-arena__dmg') : 'vpet-arena__dmg is-miss'}>{dmg > 0 ? `−${dmg}` : t('duel.dodge')}</span>}
        {live && !attack && turn.by === i && <span key={`b${n}`} className={`vpet-arena__aura is-${skill}`} aria-hidden='true'>{fx}</span>}
      </div>
    );
  };
  return (
    <div className='vpet-arena'>
      <header className='vpet-battle-head'><div><b>{t('duel.title')}</b><small>{`${duel.a.name} đối đầu ${duel.b.name}`}</small></div><span>{done ? t('duel.finished') : `Lượt ${n}/${duel.turns.length}`}</span></header>
      <div className='vpet-arena__stage' style={{'--arena-tiles': `url(${tiles})`}}>
        <div className='vpet-arena__sky'/>
        <div className='vpet-arena__floor'/>
        {fighter(foe, 'far')}
        {fighter(mine, 'near')}
        {info(foe, 'far')}
        {info(mine, 'near')}
        {attack && (
          <span key={`p${n}`} className={`vpet-arena__shot ${turn.by === mine ? 'is-up' : 'is-down'}`} style={{'--shot': SHOT_COLOR[skill] || '#fff'}} aria-hidden='true'>
            <i/>{fx}
          </span>
        )}
        {attack && dmg > 0 && <div key={`f${n}`} className={crit ? 'vpet-arena__flash is-crit' : 'vpet-arena__flash'}/>}
      </div>
      {turn?.sudden && <p className='vpet-note' role='status'>{t('duel.suddenNote')}</p>}
      <p className='vpet-arena__log' role='status' aria-label={log}><Typed text={log}/></p>
      <div className='vpet-arena__btns'>
        {!done && <button type='button' className='vpet-btn' onClick={() => setPaused(!paused)}>{paused ? 'Tiếp tục' : 'Tạm dừng'}</button>}
        <button type='button' className='vpet-btn' onClick={() => setSpeed(speed === 1 ? 2 : 1)} aria-label={t('duel.speed')}>{`${speed}×`}</button>
        {!done && <button type='button' className='vpet-btn' onClick={() => setN(duel.turns.length)}>{t('duel.skip')}</button>}
        {done && <button type='button' className='vpet-btn' onClick={() => { setPaused(false); setN(0); }}>{t('duel.replay')}</button>}
        <button type='button' className='vpet-btn' onClick={onClose}>{t('app.close')}</button>
      </div>
    </div>
  );
}

// The arena. Every colleague with a pet is ranked by arena points; power only explains
// matchup difficulty and never decides rank.
function Fighter({row, state, busy, pet, onChallenge}) {
  const gap = pet ? row.power - (state.power?.[state.pets.indexOf(pet)] || 0) : 0;
  const tone = gap > 25 ? 'is-hard' : gap < -25 ? 'is-easy' : 'is-even';
  return (
    <li className={`vpet-arena__row ${row.me ? 'is-me' : ''}`}>
      <span className='vpet-arena__rank' aria-hidden='true'>{row.badge}</span>
      <div className='vpet-arena__who'>
        <b>{row.pet}</b>
        <small>{`${row.name} · Lv ${row.lv}`}</small>
      </div>
      <div className='vpet-arena__record'>
        <b>{`${row.points} điểm`}</b>
        <small>{`${row.wins}T / ${row.losses}B`}</small>
        <small className={`vpet-arena__gap ${tone}`}>{`Lực ${row.power}${row.me || !pet ? '' : gap > 0 ? ` (+${gap})` : ` (${gap})`}`}</small>
      </div>
      {row.me ? <span className='vpet-arena__you'>{t('duel.you')}</span> : (
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !pet || !row.ready}
          title={row.ready ? '' : t('duel.unfit')}
          onClick={() => onChallenge(row.user)}>{row.ready ? t('duel.challenge') : t('duel.resting')}</button>
      )}
    </li>
  );
}

export default function DuelTab({state, sel, busy, act, replay, onReplayed}) {
  const [to, setTo] = useState('');
  const [watch, setWatch] = useState(null);
  const [view, setView] = useState(state.din?.length ? 'invites' : 'board');
  useEffect(() => {
    const d = replay && (state.duels || []).find((x) => x.id === replay);
    if (d) {
      setWatch(d);
      onReplayed();
    }
  }, [replay, state.duels]);
  const pet = state.pets[sel];
  const incoming = state.din || [];
  const waiting = state.dout || [];
  const history = state.duels || [];
  const board = (state.arena || []).map((row) => ({...row, badge: ['🥇', '🥈', '🥉'][row.rank - 1] || row.rank}));

  if (watch) {
    return <Replay duel={watch} state={state} onClose={() => setWatch(null)}/>;
  }
  const accept = async (id) => {
    const next = await act(() => api.duelAction(id, 'accept', sel), true);
    if (next?.duels?.[0]?.id === id) {
      setWatch(next.duels[0]);
    }
  };
  const challenge = async (who) => {
    const next = await act(() => api.challenge(who, sel));
    if (next) setTo('');
  };
  const TABS = [['board', t('duel.board', board.length)], ['invites', t('duel.invites', incoming.length + waiting.length)], ['history', t('duel.history', history.length)]];

  return (
    <div className='vpet-duel'>
      <header className='vpet-duel__hero'>
        {pet ? (
          <>
            <PetCanvas pet={pet} parts={pet.parts} scale={2} still={true}/>
            <div className='vpet-duel__me'>
              <b>{pet.name}</b>
              <small>{`Lv ${pet.lv || 1} · ${pet.arena_points || 1000} điểm · ${pet.wins || 0} thắng / ${pet.losses || 0} thua`}</small>
              <small>{skillText(state, pet.skills) || t('duel.noSkill')}</small>
              {(pet.dead || pet.sick || pet.health < 30) && <small role='status'>{t('duel.needCare')}</small>}
            </div>
            <div className='vpet-duel__power'>
              <span>{t('duel.power')}</span>
              <b>{state.power?.[sel] || 0}</b>
            </div>
          </>
        ) : <p className='vpet-muted'>{t('duel.noPet')}</p>}
      </header>
      <details className='vpet-duel__rules'>
        <summary>{t('duel.rules')}</summary>
        <p>{t('duel.warning')}</p>
        <p>{t('duel.rankRule')}</p>
      </details>
      <nav className='vpet-inv__tabs vpet-duel__tabs' aria-label='Khu vực đấu trường'>
        {TABS.map(([id, label]) => (
          <button key={id} type='button' aria-pressed={view === id} className={view === id ? 'is-active' : ''} onClick={() => setView(id)}>{label}</button>
        ))}
      </nav>
      {view === 'board' && (
        <section aria-label='Bảng xếp hạng'>
          {board.length ? (
            <ul className='vpet-arena__board'>
              {board.map((row) => <Fighter key={row.user} row={row} state={state} busy={busy} pet={pet && !pet.dead && !pet.sick && pet.health >= 30 ? pet : null} onChallenge={challenge}/>)}
            </ul>
          ) : <p className='vpet-empty'>{t('duel.emptyBoard')}</p>}
          <details className='vpet-duel__byname'>
            <summary>{t('duel.byName')}</summary>
            <UserPicker label='Đối thủ' value={to} onChange={setTo}/>
            <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !to.trim() || !pet || pet.dead} onClick={() => challenge(to)}>{t('duel.send')}</button>
          </details>
        </section>
      )}
      {view === 'invites' && (
        <section aria-label='Lời mời'>
          {incoming.length > 0 && <h4>{t('duel.incoming')}</h4>}
          {incoming.map((c) => (
            <div key={c.id} className='vpet-offer'>
              {c.pet_view && <PetCanvas pet={{acc: c.pet_view.acc}} parts={c.pet_view.parts} scale={1.5} still={true}/>}
              <p>{`${c.from_name} thách đấu bằng ${c.pet_view?.name || '?'}`}<br/><small>{skillText(state, c.pet_view?.skills)}</small></p>
              <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !pet || pet.dead} onClick={() => accept(c.id)}>{`⚔️ Nhận lời${pet ? ` (${pet.name})` : ''}`}</button>
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.duelAction(c.id, 'decline'))}>{t('duel.decline')}</button>
            </div>
          ))}
          {waiting.length > 0 && <h4>{t('duel.waiting')}</h4>}
          {waiting.map((c) => (
            <div key={c.id} className='vpet-offer'>
              <p>{`${c.pet_view?.name || '?'} thách đấu ${c.to_name}`}</p>
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => act(() => api.duelAction(c.id, 'cancel'))}>{t('duel.cancel')}</button>
            </div>
          ))}
          {!incoming.length && !waiting.length && <p className='vpet-empty'>{t('duel.noInvites')}</p>}
        </section>
      )}
      {view === 'history' && (
        <section aria-label='Trận đã đấu'>
          {history.length ? history.map((d) => {
            const meA = state.pets.some((p) => p.born === d.a.born) && d.a.user !== d.b.user;
            const mine = meA ? 0 : 1;
            return (
              <div key={d.id} className='vpet-offer'>
                <p>{`${d.winner === mine ? '🏆' : '💢'} ${d.a.name} ⚔️ ${d.b.name}`}<br/><small>{new Date(d.at).toLocaleString('vi-VN')}</small></p>
                <button type='button' className='vpet-btn' onClick={() => setWatch(d)}>{'▶ Xem lại'}</button>
              </div>
            );
          }) : <p className='vpet-empty'>{t('duel.noHistory')}</p>}
        </section>
      )}
    </div>
  );
}
