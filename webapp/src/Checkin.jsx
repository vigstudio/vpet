import React from 'react';

import * as api from './api';
import {t} from './i18n';
import ItemIcon from './ItemIcon';
import {itemOf} from './store';

// The attendance calendar opens by itself once a day: the whole month is visible, so the
// player can see what a long run is worth before deciding to keep coming back.
export default function Checkin({state, busy, act, onClose}) {
  const c = state.checkin;
  if (!c) return null;
  const [year, month] = c.month.split('-').map(Number);
  // a Vietnamese calendar starts on Monday
  const lead = (new Date(year, month - 1, 1).getDay() + 6) % 7;
  const claim = async () => {
    const next = await act(() => api.checkin());
    if (next) onClose();
  };
  return (
    <div className='vpet-checkin' role='dialog' aria-label={t('checkin.title')}>
      <div className='vpet-checkin__card'>
        <header>
          <div>
            <b>{t('checkin.title')}</b>
            <small>{t('checkin.sub', month, year, c.streak, c.best)}</small>
          </div>
          <button type='button' className='vpet-iconbtn' aria-label={t('app.close')} onClick={onClose}>{'✕'}</button>
        </header>
        <div className='vpet-checkin__week' aria-hidden='true'>
          {['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'].map((d) => <span key={d}>{t(`week.${d}`)}</span>)}
        </div>
        <ol className='vpet-checkin__grid'>
          {Array.from({length: lead}, (_, i) => <li key={`pad-${i}`} className='is-pad' aria-hidden='true'/>)}
          {c.days.map((d) => {
            const it = itemOf(d.item);
            const cls = ['vpet-checkin__day', d.today ? 'is-today' : '', d.claimed ? 'is-claimed' : '', d.past && !d.claimed ? 'is-past' : ''].filter(Boolean).join(' ');
            return (
              <li key={d.day} className={cls}>
                <b>{d.day}</b>
                <ItemIcon id={d.item} size={22}/>
                <small title={it?.name}>{it?.name}</small>
                {d.claimed && <span className='vpet-checkin__tick' aria-label={t('checkin.got')}>{'✓'}</span>}
              </li>
            );
          })}
        </ol>
        <ul className='vpet-checkin__milestones'>
          {c.milestones.map((m) => {
            const it = itemOf(m.item);
            return (
              <li key={m.days} className={m.reached ? 'is-reached' : ''}>
                <b>{t('checkin.days', m.days)}</b>
                <ItemIcon id={m.item} size={20}/>
                <small>{`${it?.name} + ${m.coin} 🪙`}</small>
              </li>
            );
          })}
        </ul>
        <footer>
          <small>{c.claimed_today ? t('checkin.done') : t('checkin.todo')}</small>
          <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || c.claimed_today} onClick={claim}>
            {c.claimed_today ? t('checkin.claimed') : t('checkin.claim')}
          </button>
        </footer>
      </div>
    </div>
  );
}
