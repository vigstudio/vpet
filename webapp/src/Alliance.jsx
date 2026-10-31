import React, {useState} from 'react';

import * as api from './api';
import {t} from './i18n';
import {useNow} from './store';
import {Replay} from './Duel';
import Realm from './Realm';
const {influenceFor, leaderOf, placeOf, rankRows, roadsOf, stateOf, warCountdown} = require('./alliance.cjs');

const ROLE = {
  leader: 'alliance.roleLeader',
  officer: 'alliance.roleOfficer',
  member: 'alliance.roleMember',
};

const CREST = ['⚔️', '🛡️', '🔥', '🐉', '🦅', '🌟', '⚜️', '🗡️'];
const crestOf = (id = '') => CREST[[...id].reduce((n, c) => n + c.charCodeAt(0), 0) % CREST.length];

// A territory is a castle on the war map: a banner for whoever holds it, a ring that fills
// with the influence spent on it today, and the flag of the alliance that is ahead.
function Castle({land, i, mineID, picked, onPick}) {
  const side = stateOf(land, mineID);
  const ahead = leaderOf(land);
  const need = 200 + 50 * (land.tier || 1);
  const at = placeOf(land, i);
  const held = land.owner_tag ? `[${land.owner_tag}] ${land.owner_name}` : t('alliance.neutral');
  return (
    <button
      type='button'
      className={`vpet-castle is-${side}${land.target ? ' is-target' : ''}${picked ? ' is-picked' : ''}`}
      style={{
        left: `${at.left}%`,
        top: `${at.top}%`,
        '--fill': `${Math.min(100, Math.round(((ahead?.points || 0) / need) * 100))}%`,
      }}
      aria-label={`${land.name}, ${t('alliance.tier', land.tier)}, ${held}, ${t('alliance.influence', influenceFor(land, mineID))}`}
      aria-pressed={picked}
      onClick={() => onPick(land.id)}
    >
      <span className='vpet-castle__flag' aria-hidden='true'>{land.owner ? crestOf(land.owner) : '🏳️'}</span>
      <span className='vpet-castle__keep' aria-hidden='true' data-tier={land.tier}/>
      <span className='vpet-castle__name'>{land.name}</span>
      {land.target && <span className='vpet-castle__order' aria-hidden='true'>{'⚔'}</span>}
      {land.held_days > 0 && <span className='vpet-castle__days' aria-hidden='true'>{`${land.held_days}d`}</span>}
    </button>
  );
}

// The order panel under the map: what you can do to the castle you picked, next to the
// numbers that decide whether it is worth doing.
function Orders({land, mineID, war, busy, run, pet, canManage}) {
  if (!land) {
    return <p className='vpet-war__pick'>{t('alliance.pickLand')}</p>;
  }
  const ahead = leaderOf(land);
  const need = 200 + 50 * (land.tier || 1);
  const held = land.owner_tag ? `[${land.owner_tag}] ${land.owner_name}` : t('alliance.neutral');
  return (
    <div className={`vpet-orders is-${stateOf(land, mineID)}`}>
      <header>
        <span className='vpet-orders__crest' aria-hidden='true'>{land.owner ? crestOf(land.owner) : '🏳️'}</span>
        <div>
          <b>{land.name}</b>
          <small>{`${t('alliance.tier', land.tier)} · ${held}${land.held_days ? ` · ${t('alliance.heldDays', land.held_days)}` : ''}`}</small>
        </div>
      </header>
      <dl className='vpet-orders__stats'>
        <div><dt>{t('alliance.yourInfluence')}</dt><dd>{influenceFor(land, mineID)}</dd></div>
        <div><dt>{t('alliance.leading')}</dt><dd>{ahead ? `${ahead.points}` : '—'}</dd></div>
        <div><dt>{t('alliance.needed')}</dt><dd>{need}</dd></div>
      </dl>
      <div className='vpet-orders__acts'>
        {canManage && land.can_target && <button type='button' className='vpet-btn' disabled={busy} onClick={() => run('target', {id: land.id})}>{t('alliance.target')}</button>}
        {land.target && <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || war.attacks_left <= 0} onClick={() => run('attack', {id: land.id, pet})}>{t('alliance.attack', war.attacks_left)}</button>}
        {canManage && land.can_abandon && <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy} onClick={() => run('abandon', {id: land.id})}>{t('alliance.abandon')}</button>}
        {!land.target && !land.can_target && <small className='vpet-orders__why'>{t('alliance.notTargetable')}</small>}
      </div>
    </div>
  );
}

function Board({title, rows, render, isMe}) {
  return (
    <section>
      <h4>{title}</h4>
      <ol>
        {rows.slice(0, 10).map((row) => {
          const cell = render(row);
          return (
            <li key={cell.key} className={isMe(row) ? 'is-me' : ''}>
              <i aria-hidden='true'>{['🥇', '🥈', '🥉'][row.rank - 1] || row.rank}</i>
              <b>{cell.name}</b>
              <span>{cell.note}</span>
            </li>
          );
        })}
        {!rows.length && <li className='is-empty'>{t('alliance.noRanks')}</li>}
      </ol>
    </section>
  );
}

function Leaderboards({war, mineID, me}) {
  return (
    <div className='vpet-war__rankings'>
      <Board
        title={t('alliance.rankAlliance')}
        rows={rankRows(war.alliance_ranks)}
        isMe={(row) => row.alliance === mineID}
        render={(row) => ({key: row.alliance, name: `[${row.tag}] ${row.name}`, note: t('alliance.rankPoints', row.points, row.lands)})}
      />
      <Board
        title={t('alliance.rankPersonal')}
        rows={rankRows(war.personal_ranks)}
        isMe={(row) => row.user === me}
        render={(row) => ({key: row.user, name: row.name, note: t('alliance.contribution', row.points, row.wins)})}
      />
    </div>
  );
}

function Reward({war, busy, run}) {
  if (!war.reward) {
    return null;
  }
  return (
    <div className={`vpet-war__reward${war.reward.claimed ? ' is-claimed' : ''}`}>
      <span className='vpet-war__medal' aria-hidden='true'>{'🏆'}</span>
      <div><b>{t('alliance.reward', war.reward.bps / 100)}</b><small>{t('alliance.rewardSub')}</small></div>
      <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || war.reward.claimed} onClick={() => run('claim-season')}>
        {war.reward.claimed ? t('alliance.claimed') : t('alliance.claim')}
      </button>
    </div>
  );
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
  const [pick, setPick] = useState('');
  const [tab, setTab] = useState('war');
  const run = (action, payload = {}) => act(() => api.alliance(action, payload)).then((next) => {
    if (next?.war_result) {
      setReplay(next.war_result);
    }
    return next;
  });
  const canManage = mine?.role === 'leader' || mine?.role === 'officer';

  if (replay) {
    return <Replay duel={replay} state={state} onClose={() => setReplay(null)} onComplete={() => setReplay(null)}/>;
  }

  // Not in an Alliance yet: a recruiting hall under the mountains, not a form on a blank page.
  if (!mine) {
    const locked = Math.max(0, Math.ceil(((alliance.locked_until || 0) - now) / 3600000));
    return (
      <section className='vpet-alliance is-outside'>
        <Realm id='nui' className='vpet-alliance__gate'>
          <div className='vpet-realm__heading'><b>{t('alliance.title')}</b><small>{t('alliance.intro')}</small></div>
        </Realm>
        <Reward war={war} busy={busy} run={run}/>
        {locked > 0 && <p className='vpet-alliance__lock'>{t('alliance.locked', locked)}</p>}
        <div className='vpet-hall'>
          <h4>{t('alliance.available')}</h4>
          <div className='vpet-hall__list'>
            {(alliance.rows || []).map((row) => (
              <article key={row.id} className={`vpet-hall__card${row.open ? ' is-open' : ''}`}>
                <span className='vpet-hall__crest' aria-hidden='true'>{crestOf(row.id)}</span>
                <div>
                  <b>{`[${row.tag}] ${row.name}`}</b>
                  <small>{t('alliance.memberCount', row.members)}</small>
                </div>
                <span className='vpet-hall__state'>{row.open ? t('alliance.open') : row.invited ? t('alliance.invited') : t('alliance.closed')}</span>
                <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || locked > 0 || row.members >= 20 || (!row.open && !row.invited)} onClick={() => run('join', {id: row.id})}>{t('alliance.join')}</button>
              </article>
            ))}
            {!alliance.rows?.length && <p className='vpet-empty'>{t('alliance.none')}</p>}
          </div>
        </div>
        <form className='vpet-hall__found' onSubmit={(event) => {event.preventDefault(); run('create', {name: name.trim(), tag: tag.trim(), open});}}>
          <h4>{t('alliance.create')}</h4>
          <p>{t('alliance.createRule')}</p>
          <div className='vpet-hall__fields'>
            <label>{t('alliance.name')}<input value={name} maxLength={24} required onChange={(event) => setName(event.target.value)}/></label>
            <label>{t('alliance.tag')}<input value={tag} minLength={2} maxLength={5} pattern='[A-Za-z0-9]+' required onChange={(event) => setTag(event.target.value.toUpperCase())}/></label>
          </div>
          <label className='vpet-hall__check'><input type='checkbox' checked={open} onChange={(event) => setOpen(event.target.checked)}/>{t('alliance.openJoin')}</label>
          <button type='submit' className='vpet-btn vpet-btn--primary' disabled={busy || locked > 0}>{t('alliance.createAction')}</button>
        </form>
      </section>
    );
  }

  const lands = war.territories;
  const picked = lands.find((land) => land.id === pick) || lands.find((land) => land.target) || null;
  const ours = lands.filter((land) => land.mine).length;

  return (
    <section className='vpet-alliance'>
      <header className='vpet-banner'>
        <span className='vpet-banner__crest' aria-hidden='true'>{crestOf(mine.id)}</span>
        <div>
          <b>{`[${mine.tag}] ${mine.name}`}</b>
          <small>{`${t('alliance.memberCount', mine.members.length)} · ${t('alliance.landsHeld', ours)} · ${mine.open ? t('alliance.open') : t('alliance.closed')}`}</small>
        </div>
        <div className='vpet-banner__clock'>
          <b>{warCountdown(war.day_ends, now)}</b>
          <small>{t('alliance.seasonDay', war.day || 1, 14)}</small>
        </div>
      </header>

      <Reward war={war} busy={busy} run={run}/>

      <nav className='vpet-inv__tabs vpet-alliance__tabs' aria-label={t('alliance.title')}>
        {[['war', t('alliance.war')], ['ranks', t('alliance.ranks')], ['hall', t('alliance.hallTab')]].map(([id, label]) => (
          <button key={id} type='button' aria-pressed={tab === id} className={tab === id ? 'is-active' : ''} onClick={() => setTab(id)}>{label}</button>
        ))}
      </nav>

      {tab === 'war' && (
        <>
          <div className='vpet-war__map' role='group' aria-label={t('alliance.map')}>
            <svg className='vpet-war__roads' viewBox='0 0 100 100' preserveAspectRatio='none' aria-hidden='true'>
              {roadsOf(lands).map((road) => <line key={road.key} x1={road.x1} y1={road.y1} x2={road.x2} y2={road.y2}/>)}
            </svg>
            {lands.map((land, i) => (
              <Castle key={land.id} land={land} i={i} mineID={mine.id} picked={picked?.id === land.id} onPick={setPick}/>
            ))}
          </div>
          <div className='vpet-war__deck'>
            <Orders land={picked} mineID={mine.id} war={war} busy={busy} run={run} pet={sel} canManage={canManage}/>
            <div className='vpet-war__garrison'>
              <b>{t('alliance.defenceTitle')}</b>
              <small>{t('alliance.defenceSub')}</small>
              <button type='button' className={`vpet-btn${war.garrison_ready ? '' : ' vpet-btn--primary'}`} disabled={busy || !state.pets?.length} onClick={() => run('garrison', {pet: sel})}>
                {war.garrison_ready ? t('alliance.garrisonReady') : t('alliance.garrison')}
              </button>
            </div>
          </div>
          {/* The same map as a list, for screen readers and for anyone on a narrow phone. */}
          <details className='vpet-war__text'>
            <summary>{t('alliance.textMap')}</summary>
            <ul>
              {lands.map((land) => (
                <li key={land.id}>
                  <b>{land.name}</b>
                  <span>{`${t('alliance.tier', land.tier)} · ${land.owner_tag ? `[${land.owner_tag}] ${land.owner_name}` : t('alliance.neutral')} · ${t('alliance.influence', influenceFor(land, mine.id))}`}</span>
                  <button type='button' className='vpet-btn' onClick={() => setPick(land.id)}>{t('alliance.inspect')}</button>
                </li>
              ))}
            </ul>
          </details>
        </>
      )}

      {tab === 'ranks' && <Leaderboards war={war} mineID={mine.id} me={state.user}/>}

      {tab === 'hall' && (
        <div className='vpet-hall__inside'>
          {canManage && (
            <form className='vpet-hall__invite' onSubmit={(event) => {event.preventDefault(); run('invite', {to: username.trim()}).then(() => setUsername(''));}}>
              <label>{t('alliance.invite')}<input value={username} required placeholder={t('alliance.username')} onChange={(event) => setUsername(event.target.value)}/></label>
              <button type='submit' className='vpet-btn' disabled={busy}>{t('alliance.sendInvite')}</button>
              <button type='button' className='vpet-btn' disabled={busy} onClick={() => run('open', {open: !mine.open})}>{mine.open ? t('alliance.closeJoin') : t('alliance.openJoin')}</button>
            </form>
          )}
          <ul className='vpet-roster'>
            {mine.members.map((member) => (
              <li key={member.user} className={`is-${member.role}`}>
                <span className='vpet-roster__rank' aria-hidden='true'>{member.role === 'leader' ? '👑' : member.role === 'officer' ? '🎖️' : '🗡️'}</span>
                <div><b>{member.name}</b><small>{t(ROLE[member.role] || ROLE.member)}</small></div>
                {mine.role === 'leader' && member.user !== state.user && (
                  <div className='vpet-roster__acts'>
                    {member.role === 'member'
                      ? <button type='button' disabled={busy} onClick={() => run('promote', {id: member.user})}>{t('alliance.promote')}</button>
                      : <button type='button' disabled={busy} onClick={() => run('demote', {id: member.user})}>{t('alliance.demote')}</button>}
                    <button type='button' disabled={busy} onClick={() => run('transfer', {id: member.user})}>{t('alliance.transfer')}</button>
                    <button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button>
                  </div>
                )}
                {mine.role === 'officer' && member.role === 'member' && member.user !== state.user && (
                  <div className='vpet-roster__acts'><button type='button' disabled={busy} onClick={() => run('kick', {id: member.user})}>{t('alliance.kick')}</button></div>
                )}
              </li>
            ))}
          </ul>
          <footer className='vpet-hall__exit'>
            {mine.role === 'leader'
              ? <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy || mine.members.length !== 1 || lands.some((land) => land.mine)} onClick={() => run('disband')}>{t('alliance.disband')}</button>
              : <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy} onClick={() => run('leave')}>{t('alliance.leave')}</button>}
          </footer>
        </div>
      )}
    </section>
  );
}
