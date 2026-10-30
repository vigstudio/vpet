import React from 'react';

import * as api from './api';
import {t} from './i18n';

export default function Gift({state, busy, act, onClose}) {
  const gift = state.welcome_gift;
  if (!gift || gift.claimed) return null;
  const claim = async () => {
    const next = await act(() => api.claimWelcomeGift());
    if (next) onClose();
  };
  return (
    <div className='vpet-gift' role='dialog' aria-modal='true' aria-label={t('gift.title')}>
      <div className='vpet-gift__card'>
        <button type='button' className='vpet-iconbtn vpet-gift__close' aria-label={t('app.close')} onClick={onClose}>{'✕'}</button>
        <span className='vpet-gift__icon' aria-hidden='true'>{'🎁'}</span>
        <h3>{t('gift.title')}</h3>
        <strong>{t('gift.amount', gift.coin)}</strong>
        <p>{t('gift.body')}</p>
        <button type='button' className='vpet-btn vpet-btn--primary' autoFocus={true} disabled={busy} onClick={claim}>{t('gift.claim')}</button>
      </div>
    </div>
  );
}
