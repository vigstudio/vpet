import React, {useState} from 'react';

import * as api from './api';
import {tr} from './i18n';
import {Replay} from './Duel';
import Realm from './Realm';

export default function Tower({state, sel, busy, act}) {
  const [result, setResult] = useState(null);
  const [replay, setReplay] = useState(false);
  const tower = state.tower;
  const pet = state.pets?.[sel];
  if (!tower || !pet) return null;
  const enter = async () => {
    const next = await act(() => api.tower(sel, tower.next), true);
    if (next?.tower_result) {
      setResult({duel: next.tower_result, note: next.message});
      setReplay(true);
    }
  };
  if (replay && result) return <Replay duel={result.duel} state={state} onClose={() => setReplay(false)} onComplete={() => setReplay(false)}/>;
  return <section className='vpet-tower'>
    <Realm id='hang' className='vpet-tower__scene'>
      <div className='vpet-realm__heading'><b>{tr('Tháp thử thách', 'Tower of trials', '試練の塔')}</b><small>{tr(`Đã vượt ${tower.floor}/${tower.max} tầng`, `${tower.floor}/${tower.max} floors cleared`, `${tower.floor}/${tower.max}階突破`)}</small></div>
      <ol className='vpet-tower__stairs' aria-label={tr('Các mốc tháp', 'Tower milestones', '塔の節目')}>
        {[50, 40, 30, 20, 10].map((floor) => <li key={floor} className={tower.floor >= floor ? 'is-cleared' : ''}><span>{floor}</span><b>{tower.floor >= floor ? '✓' : '◆'}</b></li>)}
      </ol>
    </Realm>
    <div className='vpet-tower__encounter'>
      <div><h3>{tower.next ? tr(`Tầng ${tower.next}: ${tower.foe?.name}`, `Floor ${tower.next}: ${tower.foe?.name}`, `${tower.next}階：${tower.foe?.name}`) : tr('Đã chinh phục đỉnh tháp', 'Tower conquered', '塔を制覇しました')}</h3>
        {tower.foe && <p>{`Lv ${tower.foe.lv} · ${tr('Sức mạnh', 'Power', '戦力')} ${tower.foe.power}${tower.foe.boss ? ' · BOSS' : ''}`}</p>}
        <p>{tr(`Còn ${tower.left}/${tower.tries_per_day} lượt hôm nay`, `${tower.left}/${tower.tries_per_day} attempts today`, `本日残り${tower.left}/${tower.tries_per_day}回`)}</p>
      </div>
      <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !tower.next || !tower.left || pet.dead} onClick={enter}>{tr('Khiêu chiến', 'Challenge', '挑戦')}</button>
    </div>
    {tower.reward && <p className='vpet-tower__loot'>{tr('Thưởng vượt lần đầu', 'First-clear rewards', '初回クリア報酬')}: {`${tower.reward.coin} 🪙 · ${tower.reward.xp} XP · ${state.rarities?.[tower.reward.rarity] || ''}`}{tower.reward.gem > 0 ? tr(' · Ngọc kỹ năng và giấy đổi kỹ năng', ' · Skill stone and skill manual', ' · スキル石とスキル変更書') : ''}</p>}
    <p className='vpet-muted'>{tr('Tiến độ giữ vĩnh viễn. Thua mất 5 sức khỏe, không tử trận. Mỗi 10 tầng nhận vật phẩm kỹ năng.', 'Progress is permanent. Defeat costs 5 health, never death. Every 10 floors grants skill items.', '進行は永久保存。敗北は体力5減少、死亡なし。10階ごとにスキルアイテム獲得。')}</p>
    {result && <div className='vpet-tower__result' role='status'><b>{result.duel.winner === 0 ? tr('Vượt tầng!', 'Floor cleared!', '突破！') : tr('Chưa vượt tầng', 'Not cleared', '未突破')}</b><p>{result.note}</p></div>}
  </section>;
}
