import React from 'react';

import city from '../assets/fantasy/nav-city.png';
import pet from '../assets/fantasy/nav-pet.png';
import quest from '../assets/fantasy/nav-quest.png';
import world from '../assets/fantasy/nav-world.png';
import hunt from '../assets/fantasy/nav-hunt.png';
import bag from '../assets/fantasy/nav-bag.png';
import shop from '../assets/fantasy/nav-shop.png';
import clinic from '../assets/fantasy/nav-clinic.png';
import market from '../assets/fantasy/nav-market.png';
import duel from '../assets/fantasy/nav-duel.png';
import grave from '../assets/fantasy/nav-grave.png';
const ART = {city, pet, quest, world, hunt, bag, shop, clinic, market, duel, grave};

const PATHS = {
  city: 'M3 10 12 3l9 7M5 9v12h14V9M9 21v-7h6v7',
  pet: 'M8 14c-3 0-4 6 0 6 2 0 2-1 4-1s2 1 4 1c4 0 3-6 0-6-2-3-6-3-8 0ZM5 7v2m5-5v3m4-3v3m5 0v2',
  world: 'm3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3V6Zm6-3v15m6-12v15',
  hunt: 'M4 20 20 4M9 4h11v11M3 12a9 9 0 0 0 9 9M3 4v5h5',
  bag: 'M6 8h12l2 13H4L6 8Zm3 0V6a3 3 0 0 1 6 0v2M9 14h6',
  shop: 'M3 9 5 3h14l2 6M4 9v12h16V9M8 21v-8h5v8M3 9c0 3 4 3 4 0 0 3 5 3 5 0 0 3 5 3 5 0 0 3 4 3 4 0',
  trade: 'M3 7h16l-4-4m4 4-4 4M21 17H5l4-4m-4 4 4 4',
  market: 'M4 10h16v10a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V10Zm-1 0 2-6h14l2 6M9 21v-6h6v6M9 4v6m6-6v6',
  tag: 'M3 12V4a1 1 0 0 1 1-1h8l9 9-9 9-9-9Zm4-5h.01',
  clinic: 'M12 6v12M6 12h12M4 4h16a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1Z',
  quest: 'M9 5h6m-6 0a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2m-6 7 2 2 4-4',
  duel: 'm4 3 16 17m-3-3 3-3m-6 6 3-3M20 3 4 20m3-3-3-3m6 6-3-3',
  grave: 'M6 21V9a6 6 0 0 1 12 0v12M3 21h18M12 7v8m-3-5h6',
  settings: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM12 2v3m0 14v3M2 12h3m14 0h3M5 5l2 2m10 10 2 2M5 19l2-2M17 7l2-2',
  heart: 'M12 21 3 12C-2 5 7-1 12 6c5-7 14-1 9 6l-9 9Z',
  close: 'm5 5 14 14M19 5 5 19',
  more: 'M5 12h.01M12 12h.01M19 12h.01',
  chat: 'M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4v8Z',
};

export default function GameIcon({name, size = 20}) {
  if (ART[name]) return <img className='vpet-game-icon' src={ART[name]} width={size} height={size} alt='' aria-hidden='true'/>;
  return <svg width={size} height={size} viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.7' strokeLinecap='round' strokeLinejoin='round' aria-hidden='true'><path d={PATHS[name] || PATHS.pet}/></svg>;
}
