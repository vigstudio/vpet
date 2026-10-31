import React, {useEffect, useRef} from 'react';

import {t, tr} from './i18n';

export default function Halloween({onClose, onVisit}) {
  const dialog = useRef(null);
  useEffect(() => {
    const previous = document.activeElement;
    const node = dialog.current;
    node.showModal();
    return () => { node.close(); previous?.focus(); };
  }, []);
  return (
    <dialog ref={dialog} className='vpet-halloween' aria-labelledby='halloween-title' aria-describedby='halloween-body' onCancel={(e) => { e.preventDefault(); onClose(); }}>
      <button type='button' className='vpet-iconbtn vpet-halloween__close' aria-label={t('app.close')} onClick={onClose}>✕</button>
      <div className='vpet-halloween__moon' aria-hidden='true'>☾</div>
      <h2 id='halloween-title'>{tr('Đêm trở về', 'A night to return', '再会の夜')}</h2>
      <b>{tr('Halloween · 31/10 · giờ Việt Nam', 'Halloween · October 31 · Vietnam time', 'ハロウィン · 10月31日 · ベトナム時間')}</b>
      <p id='halloween-body'>{tr('Nghĩa trang mở nghi lễ hồi sinh. Đổi một thú đang sống để đón một người bạn đã mất trở về.', 'The graveyard opens a resurrection ritual. Exchange one living pet to bring a lost companion back.', '墓地で復活の儀式が開かれます。生きているペット1匹と引き換えに、亡くなった仲間を迎えましょう。')}</p>
      <p>{tr('Thú hiến tế mất vĩnh viễn, không thể hoàn tác. Đồ đã ghi nhận của thú hồi sinh phải còn trong kho hoặc trên thú hiến tế; nghi lễ không tạo thêm đồ.', 'The sacrificed pet is permanently lost. Recorded equipment must still be in your bag or on the sacrificed pet; the ritual creates no extra items.', '捧げたペットは永久に失われ、取り消せません。記録された装備はバッグか捧げるペットに必要です。アイテムは複製されません。')}</p>
      <small>{tr('Mộ cũ chưa lưu tiến độ: chỉ khôi phục tên và ngoại hình, bắt đầu cấp 1, không thể phục hồi đồ chưa được ghi nhận.', 'Older graves without a snapshot restore only the name and appearance at level 1, not unrecorded equipment.', '成長記録のない古いお墓は名前と外見のみ、レベル1で復活します。未記録の装備は復元できません。')}</small>
      <div className='vpet-halloween__actions'>
        <button type='button' className='vpet-btn' onClick={onClose}>{tr('Để sau', 'Later', 'あとで')}</button>
        <button type='button' className='vpet-btn vpet-btn--primary' onClick={onVisit}>{tr('Đến Nghĩa trang', 'Visit the graveyard', '墓地へ行く')}</button>
      </div>
    </dialog>
  );
}
