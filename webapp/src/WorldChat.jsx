import React, {useEffect, useRef, useState} from 'react';

import * as api from './api';
import {dateLocale, t} from './i18n';
import PetCanvas from './PetCanvas';
import {onWorldChat, receiveWorldChat, seedWorldChat, worldChatMessages} from './worldchat.cjs';

const BLOCK_KEY = 'vpet.chat.blocked';

function savedBlocked() {
  try {
    return new Set(JSON.parse(localStorage.getItem(BLOCK_KEY) || '[]'));
  } catch (e) {
    return new Set();
  }
}

function Message({message, mine, blocked, onBlock, onReport}) {
  if (blocked.has(message.account_id)) return null;
  const pet = message.pet_parts ? {parts: message.pet_parts} : null;
  const name = message.display_name || message.username;
  return (
    <li className={`vpet-chat__message${mine ? ' is-mine' : ''}`}>
      <span className='vpet-chat__avatar' aria-hidden='true'>
        {pet ? <PetCanvas pet={pet} parts={pet.parts} scale={1} still={true}/> : name.slice(0, 1).toUpperCase()}
      </span>
      <div>
        <header><b>{name}</b><time dateTime={message.created_at}>{new Date(message.created_at).toLocaleTimeString(dateLocale(), {hour: '2-digit', minute: '2-digit'})}</time></header>
        <p>{message.body}</p>
      </div>
      {!mine && (
        <details className='vpet-chat__actions'>
          <summary aria-label={t('chat.actions')}>•••</summary>
          <button type='button' onClick={() => onBlock(message.account_id, name)}>{t('chat.block')}</button>
          <button type='button' onClick={() => onReport(message.id)}>{t('chat.report')}</button>
        </details>
      )}
    </li>
  );
}

export default function WorldChat({account, compact = false, onClose}) {
  const [messages, setMessages] = useState(worldChatMessages);
  const [body, setBody] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [blocked, setBlocked] = useState(savedBlocked);
  const list = useRef(null);

  useEffect(() => {
    let alive = true;
    const off = onWorldChat((rows) => setMessages(rows));
    // The stream keeps the log current; this one fetch fills in what was said before the
    // player arrived. It runs second so it cannot overwrite a line that just streamed in.
    api.worldMessages()
      .then((rows) => { if (alive) seedWorldChat(rows); })
      .catch((e) => { if (alive) setError(e.message); });
    return () => { alive = false; off(); };
  }, []);

  useEffect(() => {
    list.current?.scrollTo({top: list.current.scrollHeight, behavior: messages.length > 1 ? 'smooth' : 'auto'});
  }, [messages]);

  const submit = async (e) => {
    e.preventDefault();
    const text = body.trim();
    if (!text || busy) return;
    setBusy(true);
    setError('');
    try {
      // Show it at once rather than waiting for the stream to echo it back; the shared log
      // drops the duplicate when it does.
      receiveWorldChat(await api.sendWorldMessage(text));
      setBody('');
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  };

  const block = (id, name) => {
    if (!window.confirm(t('chat.blockConfirm', name))) return;
    const next = new Set(blocked);
    next.add(id);
    setBlocked(next);
    try { localStorage.setItem(BLOCK_KEY, JSON.stringify([...next])); } catch (e) { /* private mode */ }
  };

  const report = async (id) => {
    if (!window.confirm(t('chat.reportConfirm'))) return;
    try {
      await api.reportWorldMessage(id);
      setError(t('chat.reported'));
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <aside className={`vpet-chat${compact ? ' is-compact' : ''}`} aria-label={t('chat.title')}>
      <header className='vpet-chat__head'>
        <div><h3>{t('chat.title')}</h3><small>{t('chat.subtitle')}</small></div>
        {onClose && <button type='button' aria-label={t('chat.close')} onClick={onClose}>×</button>}
      </header>
      <ol ref={list} className='vpet-chat__messages' aria-live='polite' aria-relevant='additions'>
        {!messages.length && !error && <li className='vpet-chat__empty'>{t('chat.empty')}</li>}
        {messages.map((message) => <Message key={message.id} message={message} mine={message.account_id === account.id} blocked={blocked} onBlock={block} onReport={report}/>)}
      </ol>
      {error && <p className='vpet-chat__error' role='status'>{error}</p>}
      <form className='vpet-chat__form' onSubmit={submit}>
        <label className='vpet-sr-only' htmlFor='world-chat-input'>{t('chat.input')}</label>
        <input id='world-chat-input' maxLength={200} value={body} onChange={(e) => setBody(e.target.value)} placeholder={t('chat.input')}/>
        <button type='submit' disabled={busy || !body.trim()}>{busy ? t('chat.sending') : t('chat.send')}</button>
      </form>
    </aside>
  );
}
