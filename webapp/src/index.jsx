import React, {useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';

import * as api from './api';
import Panel from './Panel';
import Profile from './Profile';
import {setLocale} from './i18n';
import {openAt, refresh, say, setUnauthorizedHandler} from './store';
import {lastWorldChatID, receiveWorldChat} from './worldchat.cjs';
import './style.css';
import './shell.css';

const AUTH_TEXT = {
  vi: {title: 'Thú ảo Thiên Đảo', intro: 'Nuôi thú, phiêu lưu và giao thương trong thế giới riêng.', tabs: ['Đăng nhập', 'Tạo mới', 'Chuyển dữ liệu'], method: 'Cách vào game', code: 'Mã chuyển một lần', username: 'Tên đăng nhập', display: 'Tên hiển thị', password: 'Mật khẩu', language: 'Ngôn ngữ', busy: 'Đang xử lý…', submit: ['Vào game', 'Tạo tài khoản', 'Nhận dữ liệu và vào game']},
  en: {title: 'Sky Isles Pet', intro: 'Raise pets, explore and trade in an independent world.', tabs: ['Sign in', 'Create account', 'Transfer data'], method: 'Choose how to enter', code: 'One-time transfer code', username: 'Username', display: 'Display name', password: 'Password', language: 'Language', busy: 'Working…', submit: ['Enter game', 'Create account', 'Transfer data and enter']},
  ja: {title: '天空諸島ペット', intro: '独立した世界でペットを育て、冒険し、取引しましょう。', tabs: ['ログイン', '新規登録', 'データ移行'], method: 'ゲームへの入り方', code: '一回限りの移行コード', username: 'ユーザー名', display: '表示名', password: 'パスワード', language: '言語', busy: '処理中…', submit: ['ゲームへ', 'アカウント作成', '移行してゲームへ']},
};

function Auth({onReady}) {
  const [mode, setMode] = useState('login');
  const [form, setForm] = useState({username: '', display: '', password: '', code: '', locale: 'vi'});
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const text = AUTH_TEXT[form.locale] || AUTH_TEXT.vi;
  const modeIndex = {login: 0, register: 1, claim: 2}[mode];
  const update = (key) => (e) => setForm({...form, [key]: e.target.value});
  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      let account;
      if (mode === 'login') account = await api.login(form.username, form.password);
      if (mode === 'register') account = await api.register(form.username, form.display, form.password, form.locale);
      if (mode === 'claim') account = await api.claim(form.code.trim(), form.username, form.display, form.password, form.locale);
      onReady(account);
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <main className='vpet-auth'>
      <section className='vpet-auth__card' aria-labelledby='auth-title'>
        <div className='vpet-auth__mark' aria-hidden='true'>🐾</div>
        <h1 id='auth-title'>{text.title}</h1>
        <p>{text.intro}</p>
        <div className='vpet-auth__tabs' role='tablist' aria-label={text.method}>
          {['login', 'register', 'claim'].map((id, index) => <button key={id} type='button' role='tab' aria-selected={mode === id} onClick={() => { setMode(id); setError(''); }}>{text.tabs[index]}</button>)}
        </div>
        <form onSubmit={submit}>
          {mode === 'claim' && <label>{text.code}<input required autoComplete='off' value={form.code} onChange={update('code')}/></label>}
          <label>{text.username}<input required minLength='3' maxLength='32' autoComplete='username' value={form.username} onChange={update('username')}/></label>
          {mode !== 'login' && <label>{text.display}<input maxLength='60' autoComplete='name' value={form.display} onChange={update('display')}/></label>}
          <label>{text.password}<input required minLength='10' maxLength='128' type='password' autoComplete={mode === 'login' ? 'current-password' : 'new-password'} value={form.password} onChange={update('password')}/></label>
          <label>{text.language}<select value={form.locale} onChange={update('locale')}><option value='vi'>Tiếng Việt</option><option value='en'>English</option><option value='ja'>日本語</option></select></label>
          {error && <p className='vpet-auth__error' role='alert'>{error}</p>}
          <button className='vpet-auth__submit' disabled={busy}>{busy ? text.busy : text.submit[modeIndex]}</button>
        </form>
      </section>
    </main>
  );
}

function Game({account, onAccount, onLogout}) {
  const [online, setOnline] = useState(false);
  useEffect(() => {
    setLocale(account.locale || 'vi');
    refresh();
    // EventSource reconnects on its own but always to the original URL, which would pin the
    // chat cursor to where this session started. Reopening by hand lets each attempt carry
    // the id actually reached, so nothing said during the gap is skipped.
    let source = null;
    let retry = 0;
    let closed = false;

    const connect = () => {
      if (closed) return;
      source = new EventSource(`/api/v1/notifications/stream?chat_after=${lastWorldChatID()}`);
      source.onopen = () => { setOnline(true); retry = 0; };
      source.onerror = () => {
        setOnline(false);
        if (closed || source.readyState !== EventSource.CLOSED) return;
        retry = Math.min(retry + 1, 5);
        setTimeout(connect, retry * 1000);
      };
      source.addEventListener('world-chat', (event) => {
        try { receiveWorldChat(JSON.parse(event.data)); } catch (e) { /* Ignore malformed chat events. */ }
      });
      source.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          refresh();
          if (data.tab) openAt(data.tab, data.duel);
          if (data.message) say(data.message);
        } catch (e) {
          // Ignore a malformed event; reconnect remains active.
        }
      };
    };

    connect();
    return () => { closed = true; source?.close(); };
  }, [account.id]);
  return <Panel account={account} profile={<Profile account={account} online={online} onAccount={onAccount} onLogout={onLogout}/>}/>;
}

function App() {
  const [account, setAccount] = useState(undefined);
  useEffect(() => {
    setUnauthorizedHandler(() => setAccount(null));
    api.setUnauthorizedHandler(() => setAccount(null));
    api.me().then(setAccount).catch(() => setAccount(null));
    return () => {
      setUnauthorizedHandler(null);
      api.setUnauthorizedHandler(null);
    };
  }, []);
  if (account === undefined) return <div className='vpet-boot'>Đang mở Thiên Đảo…</div>;
  if (!account) return <Auth onReady={setAccount}/>;
  const logout = async () => { await api.logout().catch(() => {}); setAccount(null); };
  return <Game account={account} onAccount={setAccount} onLogout={logout}/>;
}

createRoot(document.getElementById('root')).render(<App/>);
