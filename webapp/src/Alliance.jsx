import React, {useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {useNow} from './store';
import {Replay} from './Duel';
const {influenceFor, rankRows, warCountdown} = require('./alliance.cjs');

const ROLE = {
  leader: 'alliance.roleLeader',
  officer: 'alliance.roleOfficer',
  member: 'alliance.roleMember',
};

function Territory({land, mine, war, busy, run, pet, canManage}) {
  const score = influenceFor(land, mine?.id);
  const label = `${land.name}, ${t('alliance.tier', land.tier)}, ${land.owner_tag ? `[${land.owner_tag}] ${land.owner_name}` : t('alliance.neutral')}, ${t('alliance.influence', score)}`;
  return <article className={`vpet-territory ${land.mine ? 'is-mine' : ''} ${land.target ? 'is-target' : ''}`} style={{'--tx': land.x, '--ty': land.y}}>
    <button type='button' className='vpet-territory__node' aria-label={label} aria-pressed={land.target || false}>
      <b>{land.name}</b><small>{land.owner_tag ? `[${land.owner_tag}]` : t('alliance.neutral')}</small>
    </button>
    <div className='vpet-territory__info'>
      <b>{land.name}</b><span>{`${t('alliance.tier', land.tier)} · ${land.owner_tag ? `[${land.owner_tag}] ${land.owner_name}` : t('alliance.neutral')}`}</span>
      <small>{t('alliance.influence', score)}</small>
      {canManage && land.can_target && <button type='button' disabled={busy} onClick={() => run('target', {id: land.id})}>{t('alliance.target')}</button>}
      {land.target && <button type='button' disabled={busy || war.attacks_left <= 0} onClick={() => run('attack', {id: land.id, pet})}>{t('alliance.attack', war.attacks_left)}</button>}
      {canManage && land.can_abandon && <button type='button' disabled={busy} onClick={() => run('abandon', {id: land.id})}>{t('alliance.abandon')}</button>}
    </div>
  </article>;
}

function Leaderboards({war}) {
  return <div className='vpet-war__rankings'>
    <section><h4>{t('alliance.rankAlliance')}</h4><ol>{rankRows(war.alliance_ranks).slice(0, 10).map((row) => <li key={row.alliance}><b>{`${row.rank}. [${row.tag}] ${row.name}`}</b><span>{t('alliance.rankPoints', row.points, row.lands)}</span></li>)}</ol></section>
    <section><h4>{t('alliance.rankPersonal')}</h4><ol>{rankRows(war.personal_ranks).slice(0, 10).map((row) => <li key={row.user}><b>{`${row.rank}. ${row.name}`}</b><span>{t('alliance.contribution', row.points, row.wins)}</span></li>)}</ol></section>
  </div>;
}

export default function Alliance({state, sel, busy, act}) {
  const alliance = state.alliance || {rows: []};
  const mine = alliance.mine;
  const war = {territories: [], alliance_ranks: [], personal_ranks: [], ...(alliance.war || {})};
  const now = useNow();
  const [name, setName] = useState('');
  const [tag, setTag] = useState('');
  const [open, setOpen] = useState(true);
  const [username, setUsername] = useState('');
  const [replay, setReplay] = useState(null);
  const run = (action, payload = {}) => act(() => api.alliance(action, payload)).then((next) => {if (next?.war_result) setReplay(next.war_result); return next;});
  const canManage = mine?.role === 'leader' || mine?.role === 'officer';

  if (replay) return <Replay duel={replay} state={state} onClose={() => setReplay(null)}/>;

  if (!mine) {
    const locked = Math.max(0, Math.ceil(((alliance.locked_until || 0) - now) / 3600000));
    return <section className='vpet-alliance'>
      <header className='vpet-alliance__head'><div><h3>{t('alliance.title')}</h3><p>{t('alliance.intro')}</p></div></header>
      {war.reward && <div className='vpet-war__reward'><span>{t('alliance.reward', war.reward.bps / 100)}</span><button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || war.reward.claimed} onClick={() => run('claim-season')}>{war.reward.claimed ? t('alliance.claimed') : t('alliance.claim')}</button></div>}
      {locked > 0 && <p className='vpet-alliance__lock'>{t('alliance.locked', locked)}</p>}
      <div className='vpet-alliance__join'><h4>{t('alliance.available')}</h4><div className='vpet-alliance__list'>
        {(alliance.rows || []).map((row) => <article key={row.id}><div><b>{`[${row.tag}] ${row.name}`}</b><small>{t('alliance.memberCount', row.members)}</small></div><span>{row.open ? t('alliance.open') : row.invited ? t('alliance.invited') : t('alliance.closed')}</span><button type='button' className='vpet-btn' disabled={busy || locked > 0 || row.members >= 20 || (!row.open && !row.invited)} onClick={() => run('join', {id: row.id})}>{t('alliance.join')}</button></article>)}
        {!alliance.rows?.length && <p className='vpet-empty'>{t('alliance.none')}</p>}
      </div></div>
      <form className='vpet-alliance__create' onSubmit={(event) => {event.preventDefault(); run('create', {name: name.trim(), tag: tag.trim(), open});}}>
        <h4>{t('alliance.create')}</h4><p>{t('alliance.createRule')}</p>
        <label>{t('alliance.name')}<input value={name} maxLength={24} required onChange={(event) => setName(event.target.value)}/></label>
        <label>{t('alliance.tag')}<input value={tag} minLength={2} maxLength={5} pattern='[A-Za-z0-9]+' required onChange={(event) => setTag(event.target.value.toUpperCase())}/></label>
        <label className='vpet-alliance__check'><input type='checkbox' checked={open} onChange={(event) => setOpen(event.target.checked)}/>{t('alliance.openJoin')}</label>
        <button type='submit' className='vpet-btn vpet-btn--primary' disabled={busy || locked > 0}>{t('alliance.createAction')}</button>
      </form>
    </section>;
  }

  return <section className='vpet-alliance'>
    <header className='vpet-alliance__head'><div><h3>{`[${mine.tag}] ${mine.name}`}</h3><p>{t('alliance.memberCount', mine.members.length)}</p></div><span>{mine.open ? t('alliance.open') : t('alliance.closed')}</span></header>
    <section className='vpet-war'>
      <header><div><h4>{t('alliance.war')}</h4><p>{t('alliance.seasonDay', war.day || 1, 14)}</p></div><strong>{warCountdown(war.day_ends, now)}</strong></header>
      {war.reward && <div className='vpet-war__reward'><span>{t('alliance.reward', war.reward.bps / 100)}</span><button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || war.reward.claimed} onClick={() => run('claim-season')}>{war.reward.claimed ? t('alliance.claimed') : t('alliance.claim')}</button></div>}
      <div className='vpet-war__map' role='group' aria-label={t('alliance.map')}>{war.territories.map((land) => <Territory key={land.id} land={land} mine={mine} war={war} busy={busy} run={run} pet={sel} canManage={canManage}/>)}</div>
      <div className='vpet-war__territory-list'>{war.territories.map((land) => <Territory key={land.id} land={land} mine={mine} war={war} busy={busy} run={run} pet={sel} canManage={canManage}/>)}</div>
      <button type='button' className='vpet-btn' disabled={busy || !state.pets?.length} onClick={() => run('garrison', {pet: sel})}>{war.garrison_ready ? t('alliance.garrisonReady') : t('alliance.garrison')}</button>
      <Leaderboards war={war}/>
    </section>
    {canManage && <form className='vpet-alliance__invite' onSubmit={(event) => {event.preventDefault(); run('invite', {to: username.trim()}).then(() => setUsername(''));}}><label>{t('alliance.invite')}<input value={username} required placeholder={t('alliance.username')} onChange={(event) => setUsername(event.target.value)}/></label><button type='submit' className='vpet-btn' disabled={busy}>{t('alliance.sendInvite')}</button></form>}
    {canManage && <button type='button' className='vpet-btn' disabled={busy} onClick={() => run('open', {open: !mine.open})}>{mine.open ? t('alliance.closeJoin') : t('alliance.openJoin')}</button>}
    <div className='vpet-alliance__members'>{mine.members.map((member) => <article key={member.user}><div><b>{member.name}</b><small>{t(ROLE[member.role] || ROLE.member)}</small></div>{mine.role === 'leader' && member.user !== state.user && <div className='vpet-alliance__actions'>{member.role === 'member' ? <button type='button' disabled={busy} onClick={() => run('promote', {id: member.user})}>{t('alliance.promote')}</button> : <button type='button' disabled={busy} onClick={() => run('demote', {id: member.user})}>{t('alliance.demote')}</button>}<button type='button' disabled={busy} onClick={() => run('transfer', {id: member.user})}>{t('alliance.transfer')}</button><button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button></div>}{mine.role === 'officer' && member.role === 'member' && member.user !== state.user && <button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button>}</article>)}</div>
    <footer className='vpet-alliance__footer'>{mine.role === 'leader' ? <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy || mine.members.length !== 1 || war.territories.some((land) => land.mine)} onClick={() => run('disband')}>{t('alliance.disband')}</button> : <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy} onClick={() => run('leave')}>{t('alliance.leave')}</button>}</footer>
  </section>;
}
