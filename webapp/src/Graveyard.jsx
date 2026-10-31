import React, {useState} from 'react';
import * as api from './api';
import {dateLocale, tr, t} from './i18n';
import PetCanvas from './PetCanvas';
import ItemIcon from './ItemIcon';

export default function Graveyard({state, busy, act}) {
  const [chosen, setChosen] = useState('');
  const [sacrifice, setSacrifice] = useState('');
  const [armed, setArmed] = useState(false);
  const living = state.pets.filter((p) => !p.dead && !(p.care_until > state.now) && !(p.frozen_until > state.now));
  const grave = (state.graves || []).find((g) => `${g.born}:${g.died}` === chosen);
  const donor = living.find((p) => String(p.born) === sacrifice);
  const revive = async () => {
    if (!grave || !donor || busy || !armed) return;
    const next = await act(() => api.resurrect(grave.born, grave.died, donor.born));
    if (next) { setChosen(''); setSacrifice(''); setArmed(false); }
  };
  return <section className='vpet-graveyard'>
    <h3>{tr('Nghĩa trang', 'Graveyard', '墓地')}</h3>
    <p className='vpet-note'>{state.halloween ? tr('Halloween · Đổi một thú đang sống để hồi sinh một thú đã chôn. Chỉ mở ngày 31/10, giờ Việt Nam.', 'Halloween · Exchange a living pet to revive a buried pet. October 31, Vietnam time only.', 'ハロウィン · 生きているペット1体と引き換えに埋葬済みのペットを復活。ベトナム時間10月31日限定。') : tr('Nghi lễ hồi sinh mở ngày 31/10 theo giờ Việt Nam.', 'Revival opens October 31, Vietnam time.', '復活の儀式はベトナム時間10月31日に開催。')}</p>
    {!state.graves?.length && <p className='vpet-empty'>{tr('Nghĩa trang trống. Chăm thú thật tốt nhé!', 'The graveyard is empty. Take good care of your pet!', '墓地は空です。大切に育ててください！')}</p>}
    <div className='vpet-graves'>{(state.graves || []).map((g) => <article key={`${g.born}:${g.died}`} className='vpet-grave'>
      <PetCanvas parts={g.parts} pet={{}} scale={2} still/>
      <div><b>{g.name}</b><p className='vpet-muted'>{g.cause}</p><small>{new Date(g.died).toLocaleString(dateLocale())}</small>
      <p>{g.snapshot ? tr(`Hồ sơ Lv ${g.snapshot.lv || 1}, kỹ năng và trang bị đã lưu.`, `Level ${g.snapshot.lv || 1}, skills and equipment recorded.`, `Lv ${g.snapshot.lv || 1}・スキル・装備の記録あり。`) : tr('Mộ cũ chỉ còn tên và ngoại hình. Không thể khôi phục cấp, kỹ năng hay đồ lịch sử; hồi sinh cấp 1, không trang bị.', 'Legacy grave: only name and appearance remain. Revives at level 1 without historical skills or equipment.', '古い墓には名前と外見のみ。レベル1・スキルと装備なしで復活します。')}</p>
      {state.halloween && <button type='button' className='vpet-btn' disabled={busy} onClick={() => { setChosen(`${g.born}:${g.died}`); setArmed(false); }}>{tr('Chọn hồi sinh', 'Choose revival', '復活を選択')}</button>}</div>
    </article>)}</div>
    {grave && state.halloween && <div className='vpet-market__confirm'>
      <h4>{tr(`Hồi sinh ${grave.name}`, `Revive ${grave.name}`, `${grave.name}を復活`)}</h4>
      <p>{tr('Đồ đã được trả về kho khi chôn. Nghi lễ lấy lại đúng đồ, cấp cường hóa, ngọc và phụ kiện trong hồ sơ; thiếu món nào sẽ từ chối, không mất thú. Đồ của thú hiến tế được trả về kho trước.', 'Burial returned items to your bag. Revival reclaims the recorded gear, enhancement levels, gems and accessory; missing items cancel the exchange without losing a pet. Sacrifice equipment returns to your bag first.', '埋葬時の装備はバッグに返却済み。記録された強化段階・宝石・アクセサリーをバッグから回収します。不足時は交換せず、ペットも失いません。生贄の装備は先にバッグへ返却します。')}</p>
      <div className='vpet-graveyard__gear'>{Object.entries(grave.snapshot?.gear || {}).map(([slot, id]) => <span key={slot}><ItemIcon id={id} size={30}/>{`+${grave.snapshot?.gear_level?.[slot] || 0}`}</span>)}</div>
      <label>{tr('Thú hiến tế', 'Sacrifice pet', '生贄のペット')}<select value={sacrifice} disabled={busy} onChange={(e) => { setSacrifice(e.target.value); setArmed(false); }}><option value=''>{tr('Chọn thú đang sống', 'Choose a living pet', '生きているペットを選択')}</option>{living.map((p) => <option key={p.born} value={p.born}>{`${p.name} · Lv ${p.lv || 1}`}</option>)}</select></label>
      {!living.length && <p>{tr('Cần một thú đang sống để hiến tế.', 'A living pet is required.', '生きているペットが必要です。')}</p>}
      {donor && <label className='vpet-graveyard__confirm'><input type='checkbox' checked={armed} disabled={busy} onChange={(e) => setArmed(e.target.checked)}/>{tr(`Tôi đồng ý mất vĩnh viễn ${donor.name} để hồi sinh ${grave.name}. Thú hiến tế không vào nghĩa trang, không hoàn tác.`, `Permanently sacrifice ${donor.name} to revive ${grave.name}. The sacrifice leaves no grave. This cannot be undone.`, `${donor.name}を永久に失い、${grave.name}を復活させます。生贄の墓は残らず、取り消せません。`)}</label>}
      <button type='button' className='vpet-btn' disabled={busy} onClick={() => { setChosen(''); setArmed(false); }}>{t('app.cancel')}</button>
      <button type='button' className='vpet-btn vpet-btn--danger' disabled={busy || !donor || !armed} onClick={revive}>{tr('Hiến tế và hồi sinh', 'Sacrifice and revive', '生贄にして復活')}</button>
    </div>}
  </section>;
}
