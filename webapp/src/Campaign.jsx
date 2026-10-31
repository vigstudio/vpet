import React, {useCallback, useEffect, useRef, useState} from 'react';

import * as api from './api';
import Alliance from './Alliance';
import Tower from './Tower';
import {dateLocale, t, tr} from './i18n';
import {Replay} from './Duel';
import {RARITY} from './Hunt';
import ItemIcon from './ItemIcon';
import PetCanvas from './PetCanvas';
import Realm, {REALMS} from './Realm';
import WorldTab from './World';
import {itemOf, refresh, useNow} from './store';

const ROUTES = {
  vuon: [[15,65],[29,53],[43,60],[56,47],[70,39],[84,28]],
  rung: [[18,28],[28,45],[43,36],[54,57],[70,49],[82,68]],
  hang: [[15,48],[30,36],[42,58],[56,42],[69,62],[84,46]],
  nui: [[18,68],[29,50],[44,40],[58,28],[71,45],[84,22]],
  nuilua: [[16,31],[30,48],[45,32],[58,56],[72,40],[85,61]],
};
const starsText = (n) => `${'★'.repeat(n)}${'☆'.repeat(3-n)}`;

function CampaignResult({result, state, onClose, onReplay}) {
  const dialog = useRef(null);
  useEffect(() => {
    const previous = document.activeElement;
    dialog.current?.focus();
    const keyboard = (event) => {
      if (event.key === 'Escape') {
        onClose();
        return;
      }
      if (event.key !== 'Tab') return;
      const focusable = [...dialog.current.querySelectorAll('button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])')];
      if (!focusable.length) {
        event.preventDefault();
        dialog.current.focus();
      } else if (event.shiftKey && document.activeElement === focusable[0]) {
        event.preventDefault();
        focusable[focusable.length - 1].focus();
      } else if (!event.shiftKey && document.activeElement === focusable[focusable.length - 1]) {
        event.preventDefault();
        focusable[0].focus();
      }
    };
    document.addEventListener('keydown', keyboard, true);
    return () => {
      document.removeEventListener('keydown', keyboard, true);
      if (previous?.isConnected) previous.focus();
    };
  }, [onClose]);
  return <div className='vpet-result-overlay' onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
    <section ref={dialog} className='vpet-result-popup' role='dialog' aria-modal='true' aria-labelledby='vpet-result-title' tabIndex={-1}>
      <button type='button' className='vpet-result-popup__close' aria-label={t('app.close')} onClick={onClose}>{'×'}</button>
      <div className={`vpet-result-popup__mark${result.won ? '' : ' is-loss'}`} aria-hidden='true'>{result.won ? '★' : '!'}</div>
      <h3 id='vpet-result-title'>{result.won ? `${starsText(result.stars)} ${t('campaign.complete')}` : tr('Chưa vượt ải', 'Stage not cleared', 'ステージ未達成')}</h3>
      <p>{result.note}</p>
      <div className='vpet-campaign__loot'>{result.coin>0 && <span className='vpet-loot'>{`🪙 +${result.coin}`}</span>}{result.xp>0 && <span className='vpet-loot'>{`✨ +${result.xp}`}</span>}{result.finds.map((it,i)=><span key={i} className={`vpet-loot is-${RARITY[it.rarity||0]}`}><ItemIcon id={it.item} size={36}/><b>{`${itemOf(it.item)?.name}${(it.amount || 1) > 1 ? ` ×${it.amount}` : ''}`}</b><small>{result.replay_bonus ? tr('Quà chơi lại', 'Replay bonus', '再挑戦ボーナス') : (state.rarities||[])[it.rarity||0]}</small></span>)}</div>
      <div className='vpet-result-popup__actions'>
        <button type='button' className='vpet-btn' onClick={onReplay}>{t('campaign.replay')}</button>
        <button type='button' className='vpet-btn vpet-btn--primary' onClick={onClose}>{tr('Tiếp tục', 'Continue', '続ける')}</button>
      </div>
    </section>
  </div>;
}

export default function Campaign({state, sel, busy, act}) {
  const [chapter, setChapter] = useState('vuon');
  const [stage, setStage] = useState(1);
  const [difficulty, setDifficulty] = useState(0);
  const [result, setResult] = useState(null);
  const [replay, setReplay] = useState(null);
  const [view, setView] = useState('campaign');
  const now = useNow();
  const pet = state.pets[sel];
  const maps = state.maps || [];
  const rows = (state.campaign?.[sel] || []).filter((r) => r.map === chapter);
  const picked = rows.find((r) => r.stage === stage);
  const energy = state.energy?.[sel] || {now: 0, max: 0, stage_cost: 3, boss_cost: 5};
  const cost = picked?.boss ? energy.boss_cost : energy.stage_cost;
  const refill = energy.next ? Math.max(0, Math.ceil((energy.next - now)/1000)) : 0;
  const map = maps.find((m) => m.id === chapter);
  const nodes=ROUTES[chapter] || ROUTES.vuon;
  const page = Math.floor((stage - 1) / 6);
  const visibleRows = rows.slice(page * 6, page * 6 + 6);
  const closeResult = useCallback(() => setResult(null), []);
  if (replay) return <Replay duel={replay} state={state} onClose={() => setReplay(null)} onComplete={() => setReplay(null)}/>;
  const cleared = Boolean(picked?.stars[difficulty]);
  const pay = picked?.rewards?.[difficulty] || {coin: 0, xp: 0, rarity: 0, shards: 0, replay: cleared, drop_chance: 0};
  const stop = !pet ? t('campaign.needAlive') : !picked?.open ? t('campaign.locked') : difficulty > 0 && !picked.stars[difficulty-1] ? t('campaign.prevDifficulty') : pet.sick || pet.health < 30 ? t('campaign.needCare') : pet.hunger < 5 ? t('campaign.needFood') : energy.now < cost ? tr(`Thiếu năng lượng (${energy.now}/${cost})`, `Not enough energy (${energy.now}/${cost})`, `気力不足（${energy.now}/${cost}）`) : '';
  const enter = async () => {
    if (busy || stop) return;
    const next = await act(() => api.campaign(sel, chapter, stage, difficulty), true);
    if (next?.campaign_result) {setResult(next.campaign_result); setReplay(next.campaign_result.duel);}
  };
  return <div className='vpet-campaign'>
    <nav className='vpet-inv__tabs' aria-label={t('campaign.activity')}>
      {[['campaign',t('campaign.campaign')],['tower',tr('Leo tháp','Tower','塔')],['boss',t('campaign.boss')],['party',t('campaign.party')],['wild',t('campaign.free')],['alliance',t('alliance.title')]].map(([id,label]) => <button key={id} type='button' aria-pressed={view===id} className={view===id?'is-active':''} onClick={() => {setView(id); refresh();}}>{label}</button>)}
    </nav>
    {view === 'alliance' ? <Alliance state={state} sel={sel} busy={busy} act={act}/> : (!pet || pet.dead) ? <p className='vpet-empty'>{t('campaign.needAlive')}</p> : view === 'tower' ? <Tower state={state} sel={sel} busy={busy} act={act}/> : view === 'wild' ? <WorldTab state={state} sel={sel} busy={busy} act={act}/> : view === 'boss' ? <Boss state={state} sel={sel} busy={busy} act={act} now={now}/> : view === 'party' ? <Parties state={state} sel={sel} busy={busy} act={act} mapID={chapter} now={now}/> : <>
      {/* One toolbar: where you are, how hard, and what it costs. Everything else is the map. */}
      <div className='vpet-campaign__setup'>
      <div className='vpet-campaign__bar'>
        <nav className='vpet-chapters' aria-label={t('campaign.map')}>
          {maps.map((m) => {
            const progress=(state.campaign?.[sel] || []).filter((r)=>r.map===m.id);
            const got=progress.reduce((n,r)=>n+r.stars.reduce((a,b)=>a+b,0),0);
            const open=progress.some((r)=>r.open);
            return <button key={m.id} type='button' aria-pressed={chapter===m.id} className={open?'':'is-locked'}
              title={open ? `${m.name} · ${got}/${progress.length * 9} ★` : tr(`${m.name} · cần cấp ${m.min_lv}`, `${m.name} · needs level ${m.min_lv}`, `${m.name} · レベル${m.min_lv}必要`)}
              onClick={() => {setChapter(m.id);setStage(1);setDifficulty(0);setResult(null);}}>
              <span aria-hidden='true'>{m.emoji || '🗺️'}</span>
              <b>{m.name}</b>
              <small>{open ? `${got}/${progress.length * 9} ★` : `Lv ${m.min_lv}+`}</small>
              <i style={{width:`${Math.round(got/Math.max(1, progress.length * 9)*100)}%`}}/>
            </button>;
          })}
        </nav>
      <nav className='vpet-campaign__pages' aria-label={tr('Chặng chiến dịch', 'Campaign sections', '冒険の区間')}>{Array.from({length: Math.ceil(rows.length / 6)}, (_, n) => <button key={n} type='button' className='vpet-btn' aria-pressed={page === n} onClick={() => setStage(n * 6 + 1)}>{tr(`Chặng ${n + 1}`, `Section ${n + 1}`, `第${n + 1}区間`)} · {n * 6 + 1}–{Math.min(rows.length, n * 6 + 6)}</button>)}</nav>
        <div className='vpet-difficulties' role='group' aria-label={t('campaign.difficulty')}>
          {[t('difficulty.easy'),t('difficulty.medium'),t('difficulty.hard')].map((label,i)=><button key={label} type='button' aria-pressed={difficulty===i} onClick={()=>setDifficulty(i)}>{label}</button>)}
        </div>
        <div className='vpet-energy-pill' role='meter' aria-valuenow={energy.now} aria-valuemin={0} aria-valuemax={energy.max}
          aria-label={t('campaign.energy')}
          title={energy.now >= energy.max ? t('campaign.full') : tr(`Hồi 1 điểm mỗi ${Math.round((energy.tick_seconds || 360)/60)} phút · điểm kế tiếp sau ${Math.floor(refill/60)}:${String(refill%60).padStart(2,'0')}`, `1 point every ${Math.round((energy.tick_seconds || 360)/60)} min · next in ${Math.floor(refill/60)}:${String(refill%60).padStart(2,'0')}`, `${Math.round((energy.tick_seconds || 360)/60)}分ごとに1回復 · 次まで${Math.floor(refill/60)}:${String(refill%60).padStart(2,'0')}`)}>
          <i style={{width: `${energy.max ? Math.min(100, energy.now/energy.max*100) : 0}%`}}/>
          <b>{`⚡ ${energy.now}/${energy.max}`}</b>
        </div>
      </div>

      <Realm id={chapter} className='vpet-campaign__map'>
        <div className='vpet-realm__heading'><b>{map?.name}</b><small>{t('campaign.mapSub')}</small></div>
        <svg className='vpet-route' viewBox='0 0 100 100' preserveAspectRatio='none' aria-hidden='true'><polyline points={nodes.map(([x,y])=>`${x},${y}`).join(' ')}/></svg>
        {visibleRows.map((r,i)=><button key={r.id} type='button' className={`vpet-stage${r.stage===stage?' is-picked':''}${r.open?'':' is-locked'}`} style={{left:`${nodes[i][0]}%`,top:`${nodes[i][1]}%`}} aria-pressed={r.stage===stage} onClick={()=>{setStage(r.stage);setResult(null);}}><span>{r.boss?'BOSS':r.stage}</span><b>{r.name}</b><small>{r.open?starsText(r.stars[difficulty] || 0):tr(`Cấp ${r.min_lv} · khóa`, `Level ${r.min_lv} · locked`, `レベル${r.min_lv} · ロック`)}</small></button>)}
      </Realm>
      <section className='vpet-expedition-bar' aria-label={t('campaign.prepare')}>
        <div>
          <b>{`${picked?.stage || 1}. ${picked?.name || ''} · ${[t('difficulty.easy'),t('difficulty.medium'),t('difficulty.hard')][difficulty]} · ${starsText(picked?.stars[difficulty] || 0)}`}</b>
          {/* What this run costs and pays, on one line, before committing the energy to it. */}
          <span className='vpet-reward-line'>
            <span className='vpet-reward-line__cost'>{tr(`⚡ ${cost} · 🍖 5`, `⚡ ${cost} · 🍖 5`, `⚡ ${cost} · 🍖 5`)}</span>
            <b>{cleared ? t('campaign.payReplay') : t('campaign.payFirst')}</b>
            <span>{`🪙 ${pay.coin}`}</span>
            <span>{`✨ ${pay.xp}`}</span>
            {!cleared && <span className={`vpet-reward-line__drop is-${RARITY[pay.rarity]}`}>{(state.rarities || [])[pay.rarity]}</span>}
            {!cleared && pay.shards > 0 && <span>{`🔸 ×${pay.shards}`}</span>}
            {cleared && <span className='vpet-reward-line__mystery'>{tr(`Có ${Math.round(pay.drop_chance * 100)}% cơ hội rơi quà hiếm`, `${Math.round(pay.drop_chance * 100)}% chance for rare loot`, `レア報酬確率${Math.round(pay.drop_chance * 100)}%`)}</span>}
          </span>
        </div>
        <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || Boolean(stop)} onClick={enter}>{busy ? t('campaign.entering') : stop || t('campaign.enter')}</button>
      </section>
      </div>
      <details className='vpet-campaign__rules'><summary>{tr('Cách tính sao và rủi ro', 'Stars and risks', '星とリスク')}</summary><p className='vpet-muted'>{tr(`Độ khó áp dụng cho toàn bộ ${map?.name || 'bản đồ'}. 1 sao: thắng. 2 sao: còn ít nhất 35% máu. 3 sao: còn ít nhất 70% máu, thắng trong 18 lượt. Thua bị thương; vùng khó có thể tử trận. Ải thường tốn ${energy.stage_cost} năng lượng, boss chương ${energy.boss_cost}, thám hiểm ${energy.wild_cost}.`, `Difficulty applies to the entire ${map?.name || 'map'}. 1 star: win. 2 stars: at least 35% HP. 3 stars: at least 70% HP within 18 turns. Defeat causes injury; hard zones can be fatal. A stage costs ${energy.stage_cost} energy, a chapter boss ${energy.boss_cost}, exploring ${energy.wild_cost}.`, `難易度は${map?.name || 'マップ'}全体に適用。勝利で星1、HP35%以上で星2、HP70%以上かつ18ターン以内で星3。敗北すると負傷し、高難度では死亡する場合があります。気力はステージ${energy.stage_cost}、ボス${energy.boss_cost}、探索${energy.wild_cost}。`)}</p></details>
      {result && <CampaignResult result={result} state={state} onClose={closeResult} onReplay={()=>setReplay(result.duel)}/>}
    </>}
  </div>;
}

function Boss({state,sel,busy,act,now}) {
  const b=state.boss;
  if (!b) return <p className='vpet-empty'>{t('campaign.noBoss')}<button type='button' className='vpet-btn' onClick={refresh}>{t('campaign.refresh')}</button></p>;
  const wait=Math.max(0,Math.ceil((b.next-now)/1000));
  const pet=state.pets[sel];
  return <section className='vpet-boss'>
    <div className='vpet-boss__layout'>
      <div className='vpet-boss__fight'>
        <Realm id='nuilua'><div className='vpet-realm__heading'><b>{b.name}</b><small>{t('campaign.bossGlobal')}</small></div><div className='vpet-boss__beast' aria-hidden='true'><PetCanvas parts={{head:3,body:4,top:3,tail:3,eyes:5,mouth:1,c1:1,c2:2,c3:3}} pet={{}} scale={6}/></div></Realm>
        {/* A real bar: the share you personally carved out is drawn inside the damage done. */}
        <div className='vpet-bossbar' role='meter' aria-valuenow={b.hp} aria-valuemin={0} aria-valuemax={b.max_hp} aria-label={t('boss.hpLabel')}>
          <i className='vpet-bossbar__hp' style={{width:`${Math.max(0,(b.hp/b.max_hp)*100)}%`}}/>
          <i className='vpet-bossbar__mine' style={{width:`${Math.min(100,(b.mine/b.max_hp)*100)}%`}}/>
          <span>{`${b.hp.toLocaleString(dateLocale())} / ${b.max_hp.toLocaleString(dateLocale())}`}</span>
        </div>
        <div className='vpet-boss__stats'>
          <span><b>{b.fighters || 0}</b><small>{tr('Người tham chiến', 'Fighters', '参加者')}</small></span>
          <span><b>{(b.mine || 0).toLocaleString(dateLocale())}</b><small>{tr('Sát thương của bạn', 'Your damage', '自分のダメージ')}</small></span>
          <span><b>{`${b.share || 0}%`}</b><small>{tr('Đóng góp', 'Contribution', '貢献')}</small></span>
        </div>
        <div className='vpet-boss__actions'>
          <button type='button' className='vpet-btn' onClick={refresh}>{t('campaign.refresh')}</button>
          <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || b.hp<=0 || wait>0 || (pet?.lv || 1)<5} onClick={()=>act(()=>api.bossHit(sel))}>{wait?tr(`Nghỉ ${wait}s`, `Rest ${wait}s`, `休憩${wait}秒`):t('campaign.attackBoss')}</button>
          <button type='button' className='vpet-btn' disabled={busy || b.hp>0 || !b.mine || b.claimed} onClick={()=>act(()=>api.bossClaim())}>{b.claimed?t('campaign.claimed'):t('campaign.claim')}</button>
        </div>
        <small className='vpet-boss__cost'>{t('boss.cost')}</small>
        {/* Thresholds come from the server: they scale with the turnout, so printing fixed
            numbers here would be wrong the moment another player joins. */}
        <div className='vpet-boss__pay'>
          {(b.tiers || [0,0,0,0]).map((dmg,i)=>(
            <span key={i} className={(b.mine||0)>=dmg?'is-reached':''}>{t('boss.tier', dmg.toLocaleString(dateLocale()), i+1)}</span>
          ))}
        </div>
        <p className='vpet-boss__gold'>{t('boss.payNow', (b.gold || 0).toLocaleString(dateLocale()))}</p>
        <details className='vpet-campaign__rules'><summary>{tr('Luật Boss thế giới', 'World Boss rules', 'ワールドボスルール')}</summary><p className='vpet-muted'>{t('boss.note')}</p></details>
      </div>
      <aside className='vpet-boss__ranking' aria-label={tr('Bảng sát thương', 'Damage leaderboard', 'ダメージランキング')}>
        <header><div><b>{tr('Bảng sát thương', 'Damage leaderboard', 'ダメージランキング')}</b><small>{tr('Tất cả người tham chiến · cập nhật sau mỗi đòn', 'All fighters · updates after every hit', '全参加者 · 攻撃ごとに更新')}</small></div><span>{b.fighters || b.top.length}</span></header>
        {b.top.length ? <ol className='vpet-contributors'>{b.top.map((r,i)=><li key={r.user || `${r.name}-${i}`} className={i<3?`is-top-${i+1}`:''}><i>{i+1}</i><b>{r.name}</b><span>{r.damage.toLocaleString(dateLocale())}</span></li>)}</ol> : <p className='vpet-empty'>{tr('Chưa có người tham chiến.', 'No fighters yet.', 'まだ参加者はいません。')}</p>}
      </aside>
    </div>
  </section>;
}

export function Parties({state,sel,busy,act,mapID='vuon',now=Date.now()}) {
  const [map,setMap]=useState(mapID);
  const rooms=state.parties || [];
  const pet=state.pets[sel];
  const petPower=state.power?.[sel] || 0;
  const selectedMap=state.maps.find((candidate)=>candidate.id===map) || state.maps[0];
  const memberOf=(room)=>room.members.find((member)=>member.user===state.user);
  const active=rooms.find((room)=>!room.done && memberOf(room));
  const mine=rooms.filter((room)=>memberOf(room) && (!room.done || !memberOf(room).claimed));
  const available=active ? [] : rooms.filter((room)=>!room.done && !memberOf(room) && room.members.length<4);
  const create=()=>act(()=>api.party('create',sel,'',map));

  // A party is a muster board, not a form: the region it marches on behind it, four banner
  // slots that fill as people join, and one bar that says whether the march can leave.
  const PartyCard=({room})=>{
    const member=memberOf(room);
    const mapInfo=state.maps.find((candidate)=>candidate.id===room.map);
    const shownPower=room.done ? room.power : room.members.reduce((sum,row)=>sum+(row.power || 0),0);
    const projected=shownPower+(member ? 0 : petPower);
    const percent=Math.min(100,Math.round(shownPower/room.target*100));
    const enough=shownPower>=room.target;
    const expires=Math.max(0,Math.ceil((room.created+30*60*1000-now)/60000));
    const status=room.done ? (room.won ? tr('Chiến thắng','Victory','勝利') : tr('Thất bại','Defeat','敗北')) : enough ? tr('Đủ lực','Ready','戦力十分') : tr('Đang tập hợp','Recruiting','募集中');
    const rewardCoin=20+(mapInfo?.min_lv || 1)*2;
    return <article className={`vpet-muster${member?' is-mine':''}${room.done?room.won?' is-won':' is-lost':''}`} key={room.id}>
      <div className='vpet-muster__scene' style={{backgroundImage:`url(${REALMS[room.map] || REALMS.vuon})`}} aria-hidden='true'/>
      <header className='vpet-muster__head'>
        <span className={`vpet-muster__status${enough && !room.done?' is-ready':''}`}>{status}</span>
        <b>{mapInfo?.name || room.map}</b>
        <small>{room.done?tr(`${room.members.length} thành viên`,`${room.members.length} members`,`${room.members.length}人`):tr(`Còn ${expires} phút`,`Expires in ${expires} min`,`残り${expires}分`)}</small>
      </header>

      {/* The march bar: filled by everyone's power, with the threshold marked on it. */}
      <div className='vpet-muster__bar'>
        <div className='vpet-muster__gauge' role='meter' aria-valuenow={shownPower} aria-valuemin={0} aria-valuemax={room.target} aria-label={tr('Sức mạnh đội','Team power','パーティ戦力')}>
          <i style={{width:`${percent}%`}} className={enough?'is-ready':''}/>
          <b>{`${shownPower} / ${room.target}`}</b>
        </div>
        {!room.done && <small>{enough?tr('Đủ lực — chủ đội bấm Xuất phát.','Ready — the leader can depart.','戦力十分。リーダーが出発できます。'):member?tr(`Thiếu ${room.target-shownPower} lực.`,`Need ${room.target-shownPower} more power.`,`あと${room.target-shownPower}戦力。`):projected>=room.target?tr('Bạn vào là đủ lực.','Your pet makes this team ready.','参加すれば戦力が足ります。'):tr(`Vào rồi vẫn thiếu ${room.target-projected} lực.`,`Still ${room.target-projected} short after you join.`,`参加後もあと${room.target-projected}戦力。`)}</small>}
      </div>

      {/* Four banner slots. An empty one is an invitation, not a blank row. */}
      <ol className='vpet-muster__slots' aria-label={tr('Thành viên đội','Party members','メンバー')}>
        {room.members.map((row)=><li key={row.user} className={row.user===room.host?'is-host':''}>
          <span className='vpet-muster__pennant' aria-hidden='true'>{row.user===room.host?'👑':'🗡️'}</span>
          <b>{row.pet}</b>
          <small>{row.name}</small>
          <strong>{row.power || '?'}</strong>
        </li>)}
        {Array.from({length:4-room.members.length},(_,index)=><li className='is-empty' key={`empty-${index}`}>
          <span className='vpet-muster__pennant' aria-hidden='true'>{'+'}</span>
          <small>{tr('Chờ đồng đội','Open slot','空き枠')}</small>
        </li>)}
      </ol>

      {room.done && room.won && <div className='vpet-muster__spoils'><b>{tr('Quà mỗi người','Reward per player','一人分の報酬')}</b><span>{tr(`Tối đa ${rewardCoin} vàng`,`Up to ${rewardCoin} coins`,`最大${rewardCoin}コイン`)}</span><span>{`+${(mapInfo?.xp || 0)+12} XP`}</span><span>{tr('1 túi thức ăn','1 food bag','食料袋1個')}</span></div>}
      {room.done && !room.won && <p className='vpet-muster__fallen'>{tr('Đội chưa đủ lực. Thú mất 12 sức khỏe; không có quà.','Not enough power. Pets lose 12 health; no reward.','戦力不足。ペットは体力を12失い、報酬なし。')}</p>}

      <div className='vpet-muster__acts'>
        {room.done ? <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !member || member.claimed} onClick={()=>act(()=>api.party('claim',sel,room.id))}>{member?.claimed?tr('Đã nhận','Claimed','受取済み'):room.won?tr('Nhận quà','Claim reward','報酬を受取る'):tr('Xác nhận kết quả','Acknowledge result','結果を確認')}</button> : member ? <><button type='button' className='vpet-btn' disabled={busy} onClick={()=>act(()=>api.party('leave',sel,room.id))}>{room.host===state.user?tr('Giải tán đội','Disband','解散'):tr('Rời đội','Leave','退出')}</button>{room.host===state.user ? <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || room.members.length<2 || !enough} onClick={()=>act(()=>api.party('start',sel,room.id))}>{room.members.length<2?tr('Cần thêm 1 người','Need 1 more player','あと1人必要'):!enough?tr(`Thiếu ${room.target-shownPower} lực`,`Need ${room.target-shownPower} power`,`あと${room.target-shownPower}戦力`):tr('⚔ Xuất phát','⚔ Depart','⚔ 出発')}</button> : <span className='vpet-muster__waiting'>{tr('Chờ chủ đội xuất phát','Waiting for leader','リーダーを待っています')}</span>}</> : <button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || room.members.length>=4} onClick={()=>act(()=>api.party('join',sel,room.id))}>{tr('Vào đội này','Join this party','このパーティに参加')}</button>}
      </div>
    </article>;
  };

  return <section className='vpet-parties'>
    <header className='vpet-parties__hero'>
      <div><b>{tr('Săn đội','Party hunt','パーティ狩り')}</b><small>{tr('Vào đội → đủ 2 người và đủ lực → chủ đội bấm Xuất phát → nhận quà.','Join → reach 2 players and enough power → leader departs → claim rewards.','参加 → 2人以上かつ戦力達成 → リーダーが出発 → 報酬受取。')}</small></div>
      <button type='button' className='vpet-btn' onClick={refresh}>{t('campaign.refresh')}</button>
    </header>
    {mine.length>0 && <section className='vpet-party-list'><h3>{tr('Đội của bạn','Your party','自分のパーティ')}</h3>{mine.map((room)=><PartyCard room={room} key={room.id}/>)}</section>}
    {!active && available.length>0 && <section className='vpet-party-list'><h3>{tr('Đội đang chờ — vào bằng 1 nút','Open parties — join in one tap','募集中 — ワンタップ参加')}</h3>{available.map((room)=><PartyCard room={room} key={room.id}/>)}</section>}
    {!active && <section className='vpet-muster-new'>
      <header><div><b>{tr('Không thấy đội phù hợp? Tạo đội mới','No suitable party? Create one','合うパーティがない？新規作成')}</b><small>{tr(`Thú đang chọn: ${pet?.name} · Lv ${pet?.lv || 1} · Lực ${petPower}`,`Selected pet: ${pet?.name} · Lv ${pet?.lv || 1} · Power ${petPower}`,`選択中：${pet?.name} · Lv ${pet?.lv || 1} · 戦力 ${petPower}`)}</small></div></header>
      {/* Picking the region shows the region, so the choice is a place rather than a row. */}
      <div className='vpet-muster-new__maps' role='group' aria-label={tr('Chọn vùng săn','Choose hunt region','狩場を選択')}>
        {state.maps.map((candidate)=>{
          const locked=(pet?.lv || 1)<candidate.min_lv;
          return <button key={candidate.id} type='button' aria-pressed={map===candidate.id} disabled={locked} className={locked?'is-locked':''} style={{backgroundImage:`url(${REALMS[candidate.id] || REALMS.vuon})`}} onClick={()=>setMap(candidate.id)}>
            <span>
              <b>{candidate.name}</b>
              <small>{locked?tr(`Cần cấp ${candidate.min_lv}`,`Needs level ${candidate.min_lv}`,`レベル${candidate.min_lv}必要`):`${110+candidate.min_lv*12} ${tr('lực','power','戦力')}`}</small>
            </span>
          </button>;
        })}
      </div>
      <div className='vpet-muster-new__go'><div><b>{selectedMap?.desc}</b><small>{tr('Mỗi người tốn 10 độ no · đội tồn tại 30 phút','Each player costs 10 fullness · party lasts 30 minutes','一人満腹度10消費 · 30分間有効')}</small></div><button type='button' className='vpet-btn vpet-btn--primary' disabled={busy || !selectedMap || (pet?.lv || 1)<selectedMap.min_lv} onClick={create}>{tr(`Tạo đội tại ${selectedMap?.name || ''}`,`Create party at ${selectedMap?.name || ''}`,`${selectedMap?.name || ''}で作成`)}</button></div>
    </section>}
    {!active && !available.length && !mine.length && <p className='vpet-empty'>{tr('Chưa có đội nào. Tạo đội mới, đồng nghiệp sẽ thấy và vào ngay.','No parties yet. Create one so coworkers can join.','パーティはまだありません。作成すると仲間が参加できます。')}</p>}
  </section>;
}
