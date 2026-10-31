import {setLocale} from './i18n';
import {useEffect, useState} from 'react';

import * as api from './api';
import {gainedItems} from './newitems.cjs';
import {mergeFastState} from './state.cjs';
import {audio} from './audio.cjs';

// Tiny shared store: the corner widget and the panel read the same server state.
let state = null;
let fetchedAt = Date.now(); // when `state.now` was taken, so a countdown can tick on from it
let bubble = null;
let open = false;
let sel = 0; // pet shown in the panel
let jump = null; // {tab, duel} asked for by a message button
const newItems = new Set(); // quantities gained since the player last inspected each item
let unauthorized = null;
const listeners = new Set();
const emit = () => listeners.forEach((fn) => fn());

// useNow returns the server's clock carried forward locally, re-rendering once a second, so a
// countdown ticks between API calls. Always compare server timestamps against this, never
// Date.now(): the two clocks can differ by minutes.
export function useNow() {
  const [, force] = useState(0);
  useEffect(() => {
    const t = setInterval(() => force((n) => n + 1), 1000);
    return () => clearInterval(t);
  }, []);
  return (state?.now || Date.now()) + (Date.now() - fetchedAt);
}

export function setUnauthorizedHandler(fn) {
  unauthorized = fn;
}

export function useVPet() {
  const [, force] = useState(0);
  useEffect(() => {
    const fn = () => force((n) => n + 1);
    listeners.add(fn);
    return () => listeners.delete(fn);
  }, []);
  return {state, bubble, open, jump, newItems: [...newItems], sel: Math.min(sel, Math.max((state?.pets?.length || 1) - 1, 0))};
}

// say shows a speech bubble over pet `who` (and in the panel toast).
export function say(text, ms = 4000, who = sel) {
  if (!text) {
    return;
  }
  const mine = {text, who, at: Date.now()};
  bubble = mine;
  emit();
  setTimeout(() => {
    if (bubble === mine) {
      bubble = null;
      emit();
    }
  }, ms);
}

export function setOpen(value, pet) {
  open = value;
  if (pet !== undefined) {
    sel = pet;
  }
  emit();
  if (value) {
    refresh();
  }
}

// openAt opens the panel on a tab; a duel id also replays that match.
export function openAt(tab, duel) {
  if (tab) {
    jump = {tab, duel};
  }
  setOpen(true);
}

export function clearJump() {
  jump = null;
}

// The server answers in the caller's language and says which one it chose, so the client
// never has to guess from the browser.
function syncLocale(next) {
  if (next?.lang) setLocale(next.lang);
}

function rememberGains(next) {
  if (!state?.inv || !next?.inv) return;
  gainedItems(state.inv, next.inv).forEach((id) => newItems.add(id));
}

export function clearNewItem(id) {
  if (newItems.delete(id)) emit();
}

export async function refresh() {
  try {
    const next = await api.getState();
    rememberGains(next);
    state = next;
    syncLocale(state);
    fetchedAt = Date.now();
    emit();
  } catch (e) {
    if (e.status === 401) {
      state = null;
      unauthorized?.();
    }
    // Offline: keep the last state and let the next SSE/API request retry.
  }
}

// run wraps a mutating call: stores the returned state and shows its message.
// quiet keeps the result message to itself: a fight should not be spoiled before it is shown.
export async function run(call, who = sel, quiet = false) {
  try {
    const next = await call();
    rememberGains(next);
    // Fast endpoints return null for unscanned global boards. Keep the loaded rows instead of
    // making a local pet change blank the market or arena.
    state = mergeFastState(state, next);
    syncLocale(state);
    fetchedAt = Date.now();
    emit();
    if (!quiet) {
      say(next?.message, 4000, who);
      if (next?.message) audio.cue('success');
    }
    return next;
  } catch (e) {
    say(e.message, 4000, who);
    audio.cue('error');
    throw e;
  }
}

export const itemOf = (id) => state?.catalog?.find((it) => it.id === id);

export function select(i) {
  sel = i;
  emit();
}
