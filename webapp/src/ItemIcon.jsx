import React from 'react';

import sheet from '../assets/items.png';
import {ITEM_ICONS} from '../assets/items.js';
import {itemOf} from './store';

// Items that have real pixel art show it; anything else falls back to its emoji, so a new
// item works the moment it is added on the server, art or not.
const ART_ALIAS = {gem_shard: 'da', gem_atk_1: 'ngoc', gem_atk_2: 'ngoc', gem_atk_3: 'ngoc', gem_atk_4: 'ngoc', gem_def_1: 'ngoc', gem_def_2: 'ngoc', gem_def_3: 'ngoc', gem_def_4: 'ngoc', gem_hp_1: 'ngoc', gem_hp_2: 'ngoc', gem_hp_3: 'ngoc', gem_hp_4: 'ngoc', phieu_hoan_diem: 'the_quen', tinh_the_tiem_nang: 'ngoc', nuoc_tang_luc: 'vitamin', tra_sam: 'thuocbo', ruou_than: 'nuocthanh', the_nang_luong: 'the_slot'};

export default function ItemIcon({id, size = 32, className = ''}) {
  const art = ART_ALIAS[id] || id;
  const i = ITEM_ICONS.indexOf(art);
  const it = itemOf(id);
  if (i < 0) {
    return <span className={`vpet-icon is-emoji ${className}`} style={{fontSize: size * 0.8}} aria-hidden='true'>{it?.emoji || '❔'}</span>;
  }
  return (
    <span
      className={`vpet-icon ${className}`}
      style={{
        width: size,
        height: size,
        backgroundImage: `url(${sheet})`,
        backgroundSize: `${ITEM_ICONS.length * 100}% 100%`,
        backgroundPosition: `${(i / (ITEM_ICONS.length - 1)) * 100}% 0`,
      }}
      aria-hidden='true'
    />
  );
}
