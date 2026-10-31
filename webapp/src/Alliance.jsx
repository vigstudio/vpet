import React, {useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {useNow} from './store';

const ROLE = {
  leader: 'alliance.roleLeader',
  officer: 'alliance.roleOfficer',
  member: 'alliance.roleMember',
};

export default function Alliance({state, busy, act}) {
  const alliance = state.alliance || {rows: []};
  const mine = alliance.mine;
  const now = useNow();
  const [name, setName] = useState('');
  const [tag, setTag] = useState('');
  const [open, setOpen] = useState(true);
  const [username, setUsername] = useState('');
  const run = (action, payload = {}) => act(() => api.alliance(action, payload));

  if (!mine) {
    const locked = Math.max(0, Math.ceil(((alliance.locked_until || 0) - now) / 3600000));
    return <section className='vpet-alliance'>
      <header className='vpet-alliance__head'>
        <div><h3>{t('alliance.title')}</h3><p>{t('alliance.intro')}</p></div>
      </header>
      {locked > 0 && <p className='vpet-alliance__lock'>{t('alliance.locked', locked)}</p>}
      <div className='vpet-alliance__join'>
        <h4>{t('alliance.available')}</h4>
        <div className='vpet-alliance__list'>
          {(alliance.rows || []).map((row) => <article key={row.id}>
            <div><b>{`[${row.tag}] ${row.name}`}</b><small>{t('alliance.memberCount', row.members)}</small></div>
            <span>{row.open ? t('alliance.open') : row.invited ? t('alliance.invited') : t('alliance.closed')}</span>
            <button type='button' className='vpet-btn' disabled={busy || locked > 0 || row.members >= 20 || (!row.open && !row.invited)} onClick={() => run('join', {id: row.id})}>{t('alliance.join')}</button>
          </article>)}
          {!alliance.rows?.length && <p className='vpet-empty'>{t('alliance.none')}</p>}
        </div>
      </div>
      <form className='vpet-alliance__create' onSubmit={(event) => {event.preventDefault(); run('create', {name: name.trim(), tag: tag.trim(), open});}}>
        <h4>{t('alliance.create')}</h4>
        <p>{t('alliance.createRule')}</p>
        <label>{t('alliance.name')}<input value={name} maxLength={24} required onChange={(event) => setName(event.target.value)}/></label>
        <label>{t('alliance.tag')}<input value={tag} minLength={2} maxLength={5} pattern='[A-Za-z0-9]+' required onChange={(event) => setTag(event.target.value.toUpperCase())}/></label>
        <label className='vpet-alliance__check'><input type='checkbox' checked={open} onChange={(event) => setOpen(event.target.checked)}/>{t('alliance.openJoin')}</label>
        <button type='submit' className='vpet-btn vpet-btn--primary' disabled={busy || locked > 0}>{t('alliance.createAction')}</button>
      </form>
    </section>;
  }

  const canManage = mine.role === 'leader' || mine.role === 'officer';
  return <section className='vpet-alliance'>
    <header className='vpet-alliance__head'>
      <div><h3>{`[${mine.tag}] ${mine.name}`}</h3><p>{t('alliance.memberCount', mine.members.length)}</p></div>
      <span>{mine.open ? t('alliance.open') : t('alliance.closed')}</span>
    </header>
    {canManage && <form className='vpet-alliance__invite' onSubmit={(event) => {event.preventDefault(); run('invite', {to: username.trim()}).then(() => setUsername(''));}}>
      <label>{t('alliance.invite')}<input value={username} required placeholder={t('alliance.username')} onChange={(event) => setUsername(event.target.value)}/></label>
      <button type='submit' className='vpet-btn' disabled={busy}>{t('alliance.sendInvite')}</button>
    </form>}
    {canManage && <button type='button' className='vpet-btn' disabled={busy} onClick={() => run('open', {open: !mine.open})}>{mine.open ? t('alliance.closeJoin') : t('alliance.openJoin')}</button>}
    <div className='vpet-alliance__members'>
      {mine.members.map((member) => <article key={member.user}>
        <div><b>{member.name}</b><small>{t(ROLE[member.role] || ROLE.member)}</small></div>
        {mine.role === 'leader' && member.user !== state.user && <div className='vpet-alliance__actions'>
          {member.role === 'member' ? <button type='button' disabled={busy} onClick={() => run('promote', {id: member.user})}>{t('alliance.promote')}</button> : <button type='button' disabled={busy} onClick={() => run('demote', {id: member.user})}>{t('alliance.demote')}</button>}
          <button type='button' disabled={busy} onClick={() => run('transfer', {id: member.user})}>{t('alliance.transfer')}</button>
          <button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button>
        </div>}
        {mine.role === 'officer' && member.role === 'member' && member.user !== state.user && <button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button>}
      </article>)}
    </div>
    <footer className='vpet-alliance__footer'>
      {mine.role === 'leader' ? <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy || mine.members.length !== 1} onClick={() => run('disband')}>{t('alliance.disband')}</button> : <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy} onClick={() => run('leave')}>{t('alliance.leave')}</button>}
    </footer>
  </section>;
}
