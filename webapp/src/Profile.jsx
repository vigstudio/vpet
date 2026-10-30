import React, {useEffect, useRef, useState} from 'react';

import * as api from './api';
import {setLocale, t} from './i18n';
import {say} from './store';

const LANGS = [['vi', 'Tiếng Việt'], ['en', 'English'], ['ja', '日本語']];

// The account menu in the header: who you are, how you appear to other players, and the two
// account actions. The username itself is not editable — it is the login, migration claims
// are keyed on it, and the market and arena print it beside every trade and duel.
export default function Profile({account, online, onAccount, onLogout}) {
  const [open, setOpen] = useState(false);
  const [pane, setPane] = useState(''); // '', 'profile' or 'password'
  const [name, setName] = useState(account.display_name || '');
  const [lang, setLang] = useState(account.locale || 'vi');
  const [password, setPassword] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const box = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const away = (e) => { if (!box.current?.contains(e.target)) setOpen(false); };
    const key = (e) => { if (e.key === 'Escape') { setOpen(false); box.current?.querySelector('button')?.focus(); } };
    document.addEventListener('pointerdown', away);
    document.addEventListener('keydown', key);
    return () => {
      document.removeEventListener('pointerdown', away);
      document.removeEventListener('keydown', key);
    };
  }, [open]);

  const show = (next) => {
    setPane(pane === next ? '' : next);
    setNote('');
    setName(account.display_name || '');
    setLang(account.locale || 'vi');
    setPassword('');
  };

  const saveProfile = async (e) => {
    e.preventDefault();
    setBusy(true);
    setNote('');
    try {
      const next = await api.updateProfile(name.trim(), lang);
      setLocale(next.locale);
      onAccount(next);
      say(t('profile.saved'));
      setPane('');
      setOpen(false);
    } catch (err) {
      setNote(err.message);
    } finally {
      setBusy(false);
    }
  };

  const savePassword = async (e) => {
    e.preventDefault();
    setBusy(true);
    setNote('');
    try {
      await api.changePassword(password);
      say(t('profile.passwordSaved'));
      setPassword('');
      setPane('');
      setOpen(false);
    } catch (err) {
      setNote(err.message);
    } finally {
      setBusy(false);
    }
  };

  const shown = account.display_name || account.username;
  return (
    <div className='vpet-profile' ref={box}>
      <button type='button' className='vpet-profile__toggle' aria-expanded={open} aria-haspopup='menu'
        onClick={() => { setOpen(!open); setPane(''); setNote(''); }}>
        <span className={`vpet-connection ${online ? 'is-online' : ''}`} aria-hidden='true'/>
        <b>{shown}</b>
      </button>
      {open && (
        <div className='vpet-profile__menu'>
          <div className='vpet-profile__who'>
            <b>{shown}</b>
            <small>{`@${account.username}`}</small>
            <small className={online ? 'is-online' : ''}>{online ? t('profile.online') : t('profile.offline')}</small>
          </div>

          <button type='button' aria-expanded={pane === 'profile'} onClick={() => show('profile')}>{t('profile.edit')}</button>
          {pane === 'profile' && (
            <form className='vpet-profile__form' onSubmit={saveProfile}>
              <label>
                {t('profile.displayName')}
                <input maxLength={60} autoFocus={true} value={name} disabled={busy}
                  placeholder={account.username} onChange={(e) => setName(e.target.value)}/>
              </label>
              <p className='vpet-profile__hint'>{t('profile.nameHint')}</p>
              <label>
                {t('profile.language')}
                <select value={lang} disabled={busy} onChange={(e) => setLang(e.target.value)}>
                  {LANGS.map(([id, label]) => <option key={id} value={id}>{label}</option>)}
                </select>
              </label>
              <button type='submit' className='vpet-profile__save' disabled={busy}>{busy ? t('profile.saving') : t('profile.save')}</button>
            </form>
          )}

          <button type='button' aria-expanded={pane === 'password'} onClick={() => show('password')}>{t('profile.password')}</button>
          {pane === 'password' && (
            <form className='vpet-profile__form' onSubmit={savePassword}>
              <label>
                {t('profile.newPassword')}
                <input required minLength={10} maxLength={128} type='password' autoFocus={true}
                  autoComplete='new-password' value={password} disabled={busy}
                  onChange={(e) => setPassword(e.target.value)}/>
              </label>
              <button type='submit' className='vpet-profile__save' disabled={busy}>{t('profile.save')}</button>
            </form>
          )}

          {note && <p className='vpet-profile__note' role='status'>{note}</p>}
          <button type='button' className='vpet-profile__logout' onClick={onLogout}>{t('profile.logout')}</button>
        </div>
      )}
    </div>
  );
}
