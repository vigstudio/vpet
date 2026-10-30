import React, {useEffect, useRef} from 'react';

import {W, H, paint} from './sprite.cjs';
import {itemOf} from './store';

// Animated sprite: two-frame bob, occasional blink, faster frames when the pet is excited.
export default function PetCanvas({parts, pet, scale = 4, still = false, className}) {
  const ref = useRef(null);
  const worn = Object.values(pet?.gear || {}).map((id) => itemOf(id)).filter(Boolean);
  const tier = Math.max(0, ...worn.map((it) => it.rarity || 0));
  const matched = worn.filter((it) => it.rarity === tier).length >= 3;
  const effect = pet?.effect || '';
  const dead = Boolean(pet?.dead);
  const sick = Boolean(pet?.sick);
  const acc = pet?.acc || '';
  const key = JSON.stringify(parts);

  useEffect(() => {
    const ctx = ref.current?.getContext('2d');
    if (!ctx) {
      return undefined;
    }
    let frame = 0;
    const draw = () => {
      const blink = !still && frame % 14 === 13;
      paint(ctx, parts || {}, {scale, frame, effect, sick, dead, blink, still, acc});
      frame++;
    };
    draw();
    if (still || dead) {
      return undefined;
    }
    const fast = effect === 'cay' || effect === 'hyper';
    const timer = setInterval(draw, fast ? 160 : effect === 'ngu' ? 900 : 420);
    return () => clearInterval(timer);
  }, [key, effect, sick, dead, scale, still, acc]);

  return (
    <canvas
      ref={ref}
      className={[className, tier >= 2 ? `vpet-worn is-r${tier}` : '', matched ? 'is-set' : ''].filter(Boolean).join(' ')}
      width={W * scale}
      height={H * scale}
      style={{imageRendering: 'pixelated'}}
    />
  );
}
