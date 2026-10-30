import React, {useEffect, useRef, useState} from 'react';

import {searchUsers} from './api';
import {t} from './i18n';

const fullName = (u) => u.display_name || '';

// Username box that suggests colleagues by name, nickname or username while typing.
export default function UserPicker({label, value, onChange}) {
  const [users, setUsers] = useState([]);
  const [open, setOpen] = useState(false);
  const [hi, setHi] = useState(0);
  const list = useRef(null);
  const term = value.trim().replace(/^@/, '');

  useEffect(() => { list.current?.scrollIntoView?.({block: 'nearest'}); }, [users]);

  useEffect(() => {
    if (!open || !term) {
      setUsers([]);
      return undefined;
    }
    let live = true;
    const t = setTimeout(() => searchUsers(term).then((r) => live && (setUsers(r), setHi(0))).catch(() => live && setUsers([])), 200);
    return () => { live = false; clearTimeout(t); };
  }, [term, open]);

  const pick = (u) => { onChange(u.username); setOpen(false); };
  const onKey = (e) => {
    if (!open || !users.length) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      setHi((h) => (h + (e.key === 'ArrowDown' ? 1 : users.length - 1)) % users.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      pick(users[hi]);
    } else if (e.key === 'Escape') {
      e.stopPropagation();
      setOpen(false);
    }
  };

  return (
    <label className='vpet-field vpet-userpick'>
      {label}
      <input
        value={value}
        placeholder={t('userPicker.hint')}
        role='combobox'
        aria-expanded={open && users.length > 0}
        aria-autocomplete='list'
        onChange={(e) => { onChange(e.target.value); setOpen(true); }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        onKeyDown={onKey}
      />
      {open && users.length > 0 && (
        <ul ref={list} className='vpet-userpick__list' role='listbox'>
          {users.map((u, i) => (
            <li key={u.id} role='option' aria-selected={i === hi} className={i === hi ? 'is-hi' : ''} onMouseDown={(e) => { e.preventDefault(); pick(u); }} onMouseEnter={() => setHi(i)}>
              <span className='vpet-userpick__avatar' aria-hidden='true'>{(fullName(u) || u.username).slice(0, 1).toUpperCase()}</span>
              <span>{fullName(u) || u.username}</span>
              <small>{'@' + u.username}</small>
            </li>
          ))}
        </ul>
      )}
    </label>
  );
}
