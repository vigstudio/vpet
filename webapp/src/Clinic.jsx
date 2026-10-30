import React from 'react';

import * as api from './api';
import {dateLocale, t} from './i18n';
import ItemIcon from './ItemIcon';
import PetCanvas from './PetCanvas';
import {itemOf, select} from './store';

// The clinic answers one question: this pet is hurt, what can I do right now. Services come
// first because they always work; the shelf is underneath for stocking up.
export default function Clinic({state, sel, busy, act}) {
  const pet = state.pets[sel];
  // Burying the last pet leaves `sel` pointing past the end of the list, so this guard has to
  // come before anything reads the pet.
  if (!pet) return <p className='vpet-empty'>{t('pet.none')}</p>;

  const c = state.clinic || {shelf: [], cure: 0, rest: 0, revive: 0, care: []};
  const coin = state.coin || 0;
  const cared = (pet.care_until || 0) > state.now;

  const services = [
    {id: 'cure', name: t('clinic.cure'), price: c.cure, desc: t('clinic.cure.desc'),
      stop: pet.dead ? t('clinic.stop.dead') : !pet.sick ? t('clinic.stop.notSick') : ''},
    {id: 'rest', name: t('clinic.rest'), price: c.rest, desc: t('clinic.rest.desc'),
      stop: pet.dead ? t('clinic.stop.dead') : pet.health >= 100 ? t('clinic.stop.healthy') : ''},
    {id: 'revive', name: t('clinic.revive'), price: c.revive, desc: t('clinic.revive.desc'),
      stop: pet.dead ? '' : t('clinic.stop.alive')},
  ];

  return (
    <div className='vpet-clinic'>
      <p className='vpet-note'>{t('clinic.note')}</p>
      <div className='vpet-clinic__body'>
        <section className='vpet-clinic__patient' aria-label={t('clinic.patient')}>
          <header>
            <PetCanvas pet={pet} parts={pet.parts} scale={2} still={true}/>
            <div>
              <b>{pet.name}</b>
              <small>{pet.dead ? t('pet.dead') : pet.sick ? t('pet.sick') : pet.health < 40 ? t('pet.weak') : t('pet.well')}</small>
              <small>{t('pet.healthValue', Math.round(pet.health))}</small>
            </div>
          </header>
          {state.pets.length > 1 && (
            <label className='vpet-market__field'>
              {t('clinic.treatFor')}
              <select aria-label={t('clinic.choosePet')} value={sel} disabled={busy} onChange={(e) => select(Number(e.target.value))}>
                {state.pets.map((p, i) => <option key={p.born} value={i}>{`${p.name}${p.dead ? ' (đã mất)' : p.sick ? ' (bệnh)' : ''}`}</option>)}
              </select>
            </label>
          )}
          <ul className='vpet-clinic__services'>
            {services.map((s) => (
              <li key={s.id}>
                <div>
                  <b>{s.name}</b>
                  <small>{s.desc}</small>
                </div>
                <button type='button' className={`vpet-btn${!s.stop && coin >= s.price ? ' vpet-btn--primary' : ''}`}
                  disabled={busy || Boolean(s.stop) || coin < s.price}
                  onClick={() => act(() => api.treatPet(sel, s.id))}>
                  {s.stop || (coin < s.price ? t('clinic.poor', s.price) : `${s.price} 🪙`)}
                </button>
              </li>
            ))}
          </ul>
          <section className='vpet-clinic__boarding' aria-label={t('clinic.board')}>
            <div className='vpet-clinic__boarding-head'>
              <div><b>{t('clinic.board')}</b><small>{t('clinic.board.desc')}</small></div>
              {cared && <span>{t('clinic.board.until', new Date(pet.care_until).toLocaleString(dateLocale()))}</span>}
            </div>
            <div className='vpet-clinic__packages'>
              {(c.care || []).map((pkg) => (
                <button key={pkg.days} type='button' className={`vpet-btn${coin >= pkg.price && !pet.dead ? ' vpet-btn--primary' : ''}`}
                  disabled={busy || pet.dead || coin < pkg.price} onClick={() => act(() => api.boardCare(sel, pkg.days))}>
                  <b>{t('checkin.days', pkg.days)}</b><small>{coin < pkg.price ? t('clinic.board.short', pkg.price - coin) : `${pkg.price} 🪙`}</small>
                </button>
              ))}
            </div>
            <small>{cared ? t('clinic.board.extend') : t('clinic.board.intro')}</small>
          </section>
        </section>
        <section className='vpet-clinic__shelf' aria-label={t('clinic.shelf')}>
          <h4>{t('clinic.shelf')}</h4>
          <ul>
            {c.shelf.map((o) => {
              const it = itemOf(o.item);
              return (
                <li key={o.item}>
                  <ItemIcon id={o.item} size={28}/>
                  <div>
                    <b>{it?.name || o.item}</b>
                    <small>{it?.desc}</small>
                  </div>
                  <span className='vpet-clinic__price'>{`${o.price} 🪙`}</span>
                  <button type='button' className='vpet-btn' disabled={busy || coin < o.price}
                    onClick={() => act(() => api.buyCare(o.item, 1))}>{t('clinic.buy1')}</button>
                  <button type='button' className='vpet-btn' disabled={busy || coin < o.price * 5}
                    onClick={() => act(() => api.buyCare(o.item, 5))}>{t('clinic.buy5')}</button>
                </li>
              );
            })}
          </ul>
        </section>
      </div>
    </div>
  );
}

// The full picture of what a pet brings to a fight, with each number broken down so a player
// can see which choice produced it.
export function StatSheet({state, sel}) {
  const s = state.stats?.[sel];
  if (!s) return null;
  const rows = [
    {label: 'Sát thương', total: s.atk, unit: '', parts: [
      ['Cắn thường', s.base_atk], ['Trang bị, bộ đồ, ngọc', s.gear_atk], ['Điểm Công', s.build_atk], ['Danh hiệu', s.title_atk]]},
    {label: 'Giáp', total: s.def, unit: '', parts: [
      ['Trang bị, bộ đồ, ngọc', s.gear_def], ['Cấp độ', s.lv_def], ['Điểm Thủ', s.build_def], ['Danh hiệu', s.title_def]]},
    {label: 'Máu chiến đấu', total: s.hp, unit: '', parts: [
      ['Sức khỏe hiện tại', s.health_hp], ['Trang bị, bộ đồ, ngọc', s.gear_hp], ['Cấp độ', s.lv_hp], ['Điểm Sinh lực', s.build_hp], ['Danh hiệu', s.title_hp]]},
    {label: 'Tốc đánh', total: s.speed, unit: '', parts: [
      ['Nhanh hơn đối thủ thì thỉnh thoảng được đánh thêm lượt', null]]},
  ];
  return (
    <section className='vpet-statsheet' aria-label='Chỉ số chiến đấu'>
      <header>
        <b>{'Chỉ số chiến đấu'}</b>
        <span title='Số dùng để so sánh trên bảng đấu'>{`Lực chiến ${s.power}`}</span>
      </header>
      <div className='vpet-statsheet__rows'>
        {rows.map((r) => (
          <article key={r.label}>
            <div className='vpet-statsheet__total'>
              <small>{r.label}</small>
              <b>{Math.round(r.total * 10) / 10}</b>
            </div>
            <ul>
              {r.parts.filter(([, v]) => v === null || v > 0).map(([name, v]) => (
                <li key={name}><span>{name}</span>{v !== null && <b>{`+${Math.round(v * 10) / 10}`}</b>}</li>
              ))}
            </ul>
          </article>
        ))}
      </div>
    </section>
  );
}
