import React from 'react';

import city from '../assets/realm-city.webp';
import garden from '../assets/realm-vuon.webp';
import forest from '../assets/realm-rung.webp';
import cave from '../assets/realm-hang.webp';
import snow from '../assets/realm-nui.webp';
import volcano from '../assets/realm-nuilua.webp';

export const REALMS = {city, vuon: garden, rung: forest, hang: cave, nui: snow, nuilua: volcano};

// Artwork is pre-rendered from CC0 models. DOM hotspots remain keyboard-accessible; no
// WebGL, external image requests or square tile grid is shipped to the player.
export default function Realm({id = 'city', children, className = ''}) {
  return <div className={`vpet-realm ${className}`} style={{backgroundImage: `url(${REALMS[id] || city})`}}>{children}</div>;
}
