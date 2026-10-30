// 8-bit pet sprites, drawn procedurally on a 32×32 grid from the parts the user picked.
// Part counts must match partLimits in server/game.go (checked by sprite.test.cjs).
const W = 32;
const TOP = 8; // headroom above the 32×32 body grid so tall ears, horns and hats are not cut off
const H = 32 + TOP;

const COLORS = ['#ff8fb1', '#e8453c', '#ff9a3c', '#ffd23f', '#9be15d', '#3fbf6b', '#2ec4b6', '#5bc0f8', '#4064e8', '#9b5de5', '#a0663c', '#f2f2f2'];
const COLOR_NAMES = ['Hồng', 'Đỏ', 'Cam', 'Vàng', 'Chanh', 'Lá', 'Ngọc', 'Trời', 'Lam', 'Tím', 'Nâu', 'Trắng'];

const PARTS = {
  head: ['Tròn', 'Vuông', 'Bè', 'Dài', 'Hồ lô', 'Tam giác'],
  top: ['Không', 'Tai mèo', 'Tai thỏ', 'Sừng', 'Ăng-ten', 'Vương miện', 'Mầm cây', 'Mào'],
  eyes: ['Chấm', 'Long lanh', 'Lim dim', 'Một mắt', 'Ba mắt', 'Cau có', 'Đeo kính'],
  mouth: ['Cười', 'Răng nanh', 'Mỏ chim', 'Há miệng', 'Lè lưỡi', 'Răng cưa'],
  body: ['Tròn', 'Hộp', 'Quả lê', 'Que', 'Bự', 'Trứng'],
  pattern: ['Trơn', 'Đốm', 'Sọc', 'Bụng trắng', 'Trái tim'],
  legs: ['Hai chân', 'Bốn chân', 'Xúc tu', 'Lơ lửng', 'Chân chim', 'Bánh xe'],
  tail: ['Không', 'Đuôi mèo', 'Vây cá', 'Đuôi quỷ', 'Đuôi bông', 'Đôi cánh'],
  c1: COLOR_NAMES,
  c2: COLOR_NAMES,
  c3: COLOR_NAMES,
};

const PART_LABELS = {
  head: 'Đầu', top: 'Trên đầu', eyes: 'Mắt', mouth: 'Miệng', body: 'Thân', pattern: 'Hoa văn',
  legs: 'Chân', tail: 'Đuôi', c1: 'Màu chính', c2: 'Màu phụ', c3: 'Màu điểm',
};

const INK = '#1d1a2b';
const OUTLINE = '#2b2233';

function hexToRgb(h) {
  const n = parseInt(h.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function mix(a, b, t) {
  const x = hexToRgb(a);
  const y = hexToRgb(b);
  return '#' + x.map((v, i) => Math.round(v + ((y[i] - v) * t)).toString(16).padStart(2, '0')).join('');
}

function randomParts(rnd = Math.random) {
  const out = {};
  for (const k of Object.keys(PARTS)) {
    out[k] = Math.floor(rnd() * PARTS[k].length);
  }
  return out;
}

// Head shapes: [draw, top y, eye y]
const HEADS = [
  (d) => d.ell(16, 11, 7, 6, d.c1),
  (d) => d.box(9, 5, 23, 16, d.c1),
  (d) => d.ell(16, 11, 10, 5, d.c1),
  (d) => d.ell(16, 10, 5, 7, d.c1),
  (d) => { d.ell(16, 13, 7, 4, d.c1); d.ell(16, 8, 5, 4, d.c1); },
  (d) => { for (let y = 4; y <= 16; y++) { const hw = Math.round(((y - 4) * 0.75) + 1); d.rect(16 - hw, y, 16 + hw, y, d.c1); } },
];
const HEAD_TOP = [5, 5, 6, 3, 4, 4];
const HEAD_EYE = [10, 10, 11, 9, 12, 12];

const BODIES = [
  (d) => d.ell(16, 21, 6, 5, d.c1),
  (d) => d.box(10, 17, 22, 25, d.c1),
  (d) => { d.ell(16, 22, 7, 4, d.c1); d.ell(16, 18, 4, 3, d.c1); },
  (d) => d.rect(13, 16, 19, 26, d.c1),
  (d) => d.ell(16, 21, 9, 6, d.c1),
  (d) => d.ell(16, 21, 5, 6, d.c1),
];

function buildGrid(parts, opts = {}) {
  const p = {...parts};
  const frame = opts.frame || 0;
  const effect = opts.effect || '';
  const g = new Array(W * H).fill(null);
  const fx = new Array(W * H).fill(null);
  const set = (grid, x, y, c) => {
    const r = y + TOP;
    if (x >= 0 && x < W && r >= 0 && r < H) {
      grid[(r * W) + x] = c;
    }
  };
  const get = (x, y) => (x >= 0 && x < W && y >= 0 && y < H ? g[(y * W) + x] : null); // raw rows, used by finish

  if (opts.dead) {
    const d = drawer(g, set, {});
    d.ell(16, 13, 7, 5, '#9aa0a6');
    d.rect(9, 13, 23, 28, '#9aa0a6');
    d.rect(8, 28, 24, 29, '#6b7075');
    d.rect(15, 15, 17, 24, '#5f6368');
    d.rect(12, 18, 20, 19, '#5f6368');
    d.rect(10, 26, 11, 27, '#7cb342');
    d.rect(21, 25, 22, 27, '#7cb342');
    return finish(g, fx, get, set, false);
  }

  let c1 = COLORS[p.c1 || 0];
  if (effect === 'ngodoc') {
    c1 = mix(c1, '#7bd389', 0.6);
  } else if (opts.sick) {
    c1 = mix(c1, '#dfe8f0', 0.45);
  }
  const c2 = COLORS[p.c2 || 0];
  const c3 = COLORS[p.c3 || 0];
  const bob = opts.still ? 0 : frame % 2;
  const d = drawer(g, set, {c1, c2, c3});
  const head = p.head || 0;
  const ty = HEAD_TOP[head] + bob;
  const ey = HEAD_EYE[head] + bob;
  const my = ey + 3;

  // tail and wings sit behind the body
  switch (p.tail) {
  case 1: {
    const pts = [[22, 22], [23, 22], [24, 21], [25, 20], [26, 19], [26, 18], [27, 17], [27, 16], [27 + (frame % 2), 15], [26 + (frame % 2), 14]];
    pts.forEach(([x, y]) => { d.px(x, y + bob, c1); d.px(x, y + bob + 1, c1); });
    break;
  }
  case 2:
    for (let y = 17; y <= 25; y++) {
      d.rect(21, y, 21 + (y < 21 ? y - 17 : 25 - y) + 1, y, c2);
    }
    break;
  case 3: {
    const pts = [[22, 23], [23, 23], [24, 22], [25, 21], [25, 20], [26, 19], [26, 18]];
    pts.forEach(([x, y]) => d.px(x, y + bob, mix(c1, INK, 0.3)));
    d.rect(25, 16 + bob, 27, 17 + bob, c3);
    d.px(26, 15 + bob, c3);
    break;
  }
  case 4:
    d.ell(23, 23, 3, 3, mix(c3, '#ffffff', 0.4));
    break;
  case 5: {
    const ry = frame % 2 ? 3 : 5;
    d.ell(7, 17 + bob, 3, ry, mix(c2, '#ffffff', 0.35));
    d.ell(25, 17 + bob, 3, ry, mix(c2, '#ffffff', 0.35));
    break;
  }
  default:
  }

  // legs do not bob; frame alternates the stride
  const legC = mix(c1, INK, 0.25);
  const step = frame % 2;
  switch (p.legs) {
  case 1:
    [10, 14, 17, 21].forEach((x, i) => d.rect(x, 25, x + 1, ((i + step) % 2) ? 27 : 28, legC));
    break;
  case 2:
    [11, 14, 18, 21].forEach((x0) => {
      for (let y = 25; y <= 30; y++) {
        d.px(x0 + Math.round(Math.sin((y + (frame * 2) + x0) * 0.9)), y, c2);
      }
    });
    break;
  case 3:
    break;
  case 4:
    [13, 19].forEach((x, i) => {
      const end = ((i + step) % 2) ? 28 : 29;
      d.rect(x, 25, x, end, '#ff9a3c');
      d.rect(x - 1, end + 1, x + 1, end + 1, '#ff9a3c');
    });
    break;
  case 5:
    d.ell(16, 28, 3, 3, mix(c2, INK, 0.4));
    d.px(16, 28, c3);
    if (step) {
      d.px(16, 26, c3); d.px(16, 30, c3);
    } else {
      d.px(14, 28, c3); d.px(18, 28, c3);
    }
    break;
  default:
    d.rect(12, 25, 13, step ? 28 : 29, legC);
    d.rect(19, 25, 20, step ? 29 : 28, legC);
  }

  d.shift = bob;
  (BODIES[p.body] || BODIES[0])(d);
  const bodyPx = new Set();
  g.forEach((c, i) => { if (c === c1) bodyPx.add(i); });
  const onBody = (x, y, c) => { if (bodyPx.has((y * W) + x)) set(g, x, y, c); };
  switch (p.pattern) {
  case 1:
    [[12, 19], [18, 21], [15, 23], [20, 18], [11, 22], [17, 25]].forEach(([x, y]) => {
      for (let i = 0; i < 2; i++) for (let j = 0; j < 2; j++) onBody(x + i, y + j + bob, c2);
    });
    break;
  case 2:
    for (let y = 15; y < 28; y++) if ((y + bob) % 3 === 0) for (let x = 0; x < W; x++) onBody(x, y, c2);
    break;
  case 3:
    for (let y = 18; y < 27; y++) for (let x = 12; x <= 20; x++) {
      if (((x - 16) ** 2 / 12) + ((y - 22 - bob) ** 2 / 10) <= 1) onBody(x, y, mix(c1, '#ffffff', 0.55));
    }
    break;
  case 4:
    [[-2, -1], [-1, -1], [1, -1], [2, -1], [-2, 0], [-1, 0], [0, 0], [1, 0], [2, 0], [-1, 1], [0, 1], [1, 1], [0, 2]].
      forEach(([x, y]) => onBody(16 + x, 21 + y + bob, c3));
    break;
  default:
  }
  d.shift = 0;

  HEADS[head](shifted(d, bob));

  switch (p.top) {
  case 1:
    d.tri(9, ty + 1, 6, 5, c1); d.tri(17, ty + 1, 6, 5, c1);
    d.px(11, ty - 1, c3); d.px(20, ty - 1, c3);
    break;
  case 2:
    d.rect(11, ty - 7, 13, ty, c1); d.rect(12, ty - 6, 12, ty - 1, c3);
    d.rect(19, ty - 7, 21, ty, c1); d.rect(20, ty - 6, 20, ty - 1, c3);
    break;
  case 3:
    [[11, 0], [10, -1], [10, -2], [11, -3]].forEach(([x, y]) => { d.px(x, ty + y, '#f5f0e1'); d.px(32 - x, ty + y, '#f5f0e1'); });
    break;
  case 4:
    d.rect(16, ty - 4, 16, ty - 1, INK);
    d.ell(16, ty - 5, 1, 1, frame % 2 ? c3 : mix(c3, '#ffffff', 0.6));
    break;
  case 5:
    d.rect(11, ty - 2, 21, ty, '#ffd23f');
    [11, 16, 21].forEach((x) => { d.px(x, ty - 3, '#ffd23f'); d.px(x, ty - 4, '#ffd23f'); });
    d.px(16, ty - 1, c2);
    break;
  case 6:
    d.rect(16, ty - 2, 16, ty - 1, '#3fbf6b');
    [[14, -3], [15, -3], [15, -4], [17, -4], [18, -4], [17, -5]].forEach(([x, y]) => d.px(x, ty + y, '#3fbf6b'));
    break;
  case 7:
    for (let x = 13; x <= 19; x++) d.rect(x, ty - 2 - (x % 2), x, ty, c3);
    break;
  default:
  }

  // eyes (effects can override the expression)
  const eyeL = 13;
  const eyeR = 19;
  const both = (fn) => { fn(eyeL); fn(eyeR); };
  if (effect === 'ngodoc') {
    both((x) => { [[-1, -1], [1, 1], [0, 0], [1, -1], [-1, 1]].forEach(([a, b]) => d.px(x + a, ey + b, INK)); });
  } else if (effect === 'ngon' || effect === 'cuoi') {
    both((x) => { d.px(x - 1, ey, INK); d.px(x, ey - 1, INK); d.px(x + 1, ey, INK); });
  } else if (effect === 'ngu') {
    both((x) => { d.px(x - 1, ey, INK); d.px(x, ey + 1, INK); d.px(x + 1, ey, INK); });
  } else if (effect === 'dau') {
    both((x) => d.rect(x, ey, x, ey + 1, INK));
    d.px(eyeL - 1, ey - 2, INK); d.px(eyeL, ey - 1, INK); d.px(eyeR + 1, ey - 2, INK); d.px(eyeR, ey - 1, INK);
  } else if (opts.blink && p.eyes !== 2) {
    both((x) => d.rect(x - 1, ey, x + 1, ey, INK));
  } else {
    switch (p.eyes) {
    case 1:
      both((x) => { d.rect(x - 1, ey - 1, x + 1, ey + 1, '#ffffff'); d.rect(x, ey, x + 1, ey + 1, INK); });
      break;
    case 2:
      both((x) => d.rect(x - 1, ey, x + 1, ey, INK));
      break;
    case 3:
      d.ell(16, ey, 2, 2, '#ffffff'); d.rect(16, ey, 17, ey + 1, INK);
      break;
    case 4:
      [12, 20].forEach((x) => d.rect(x, ey, x, ey + 1, INK)); d.rect(16, ey - 2, 16, ey - 1, INK);
      break;
    case 5:
      both((x) => d.rect(x, ey, x, ey + 1, INK));
      d.px(12, ey - 2, INK); d.px(13, ey - 1, INK); d.px(20, ey - 2, INK); d.px(19, ey - 1, INK);
      break;
    case 6:
      both((x) => { d.rect(x - 2, ey - 1, x + 2, ey + 1, INK); d.px(x - 1, ey, '#ffffff'); d.px(x + 1, ey, '#ffffff'); d.px(x, ey, INK); });
      d.rect(15, ey - 1, 17, ey - 1, INK);
      break;
    default:
      both((x) => d.rect(x, ey, x, ey + 1, INK));
    }
  }

  let mouth = p.mouth || 0;
  if (effect === 'cay') mouth = 3;
  if (effect === 'hoi' || effect === 'dau') mouth = 5;
  if (effect === 'cuoi') mouth = 4;
  if (effect === 'ngu') mouth = 0;
  switch (mouth) {
  case 1:
    d.rect(14, my, 18, my, INK); d.px(14, my + 1, '#ffffff'); d.px(18, my + 1, '#ffffff');
    break;
  case 2:
    d.rect(15, my - 1, 17, my, '#ff9a3c'); d.px(16, my + 1, '#ff9a3c');
    break;
  case 3:
    d.rect(15, my, 17, my + 1, INK); d.px(16, my + 1, '#e8453c');
    break;
  case 4:
    d.rect(14, my, 18, my, INK); d.rect(16, my + 1, 17, my + 2, '#ff5d8f');
    break;
  case 5:
    [13, 14, 15, 16, 17, 18, 19].forEach((x, i) => d.px(x, my + (i % 2), INK));
    break;
  default:
    d.px(14, my, INK); d.rect(15, my + 1, 17, my + 1, INK); d.px(18, my, INK);
  }

  // effect particles go on top, without outline
  const f = (x, y, c) => set(fx, x, y, c);
  if (effect === 'cay' || effect === 'ngon') {
    const cheek = effect === 'cay' ? '#ff3b30' : '#ff8fb1';
    d.rect(10, my - 1, 11, my - 1, cheek); d.rect(21, my - 1, 22, my - 1, cheek);
  }
  if (effect === 'cay') {
    const flames = frame % 2 ? [[7, ty], [24, ty - 1], [6, ty + 3]] : [[8, ty - 1], [25, ty + 1], [26, ty - 2]];
    flames.forEach(([x, y]) => { f(x, y, '#ff9a3c'); f(x, y - 1, '#ffd23f'); f(x + 1, y, '#e8453c'); });
    f(23, ey - 1, '#5bc0f8'); f(23, ey, '#5bc0f8');
  } else if (effect === 'hoi') {
    for (let k = 0; k < 4; k++) {
      f(5 + ((k + frame) % 2), 13 + (k * 2), '#8bc34a');
      f(27 - ((k + frame) % 2), 12 + (k * 2), '#8bc34a');
    }
    f(4 + ((frame * 3) % 6), 6, INK); f(26 - ((frame * 2) % 5), 4 + (frame % 2), INK);
  } else if (effect === 'ngon') {
    const hx = frame % 2 ? 5 : 25;
    [[0, 0], [2, 0], [-1, 1], [0, 1], [1, 1], [2, 1], [3, 1], [0, 2], [1, 2], [2, 2], [1, 3]].forEach(([x, y]) => f(hx + x, ty - 3 + y, '#e8453c'));
  } else if (effect === 'hyper') {
    const sp = frame % 2 ? [[5, 8], [26, 14], [8, 22]] : [[25, 6], [6, 16], [24, 24]];
    sp.forEach(([x, y]) => { f(x, y, '#ffd23f'); f(x - 1, y, '#fff3b0'); f(x + 1, y, '#fff3b0'); f(x, y - 1, '#fff3b0'); f(x, y + 1, '#fff3b0'); });
  } else if (effect === 'ngodoc') {
    f(10 + (frame % 2), ty - 2, '#7bd389'); f(22 - (frame % 2), ty - 4, '#7bd389'); f(16, ty - 6 + (frame % 2), '#7bd389');
  } else if (effect === 'dau') {
    // anger mark and a plaster on the forehead
    [[0, 0], [2, 0], [1, 1], [0, 2], [2, 2]].forEach(([x, y]) => f(23 + x, ty - 1 + y + (frame % 2), '#e8453c'));
    d.rect(9, ty + 3, 12, ty + 3, '#f5d6a8'); d.px(10, ty + 2, '#f5d6a8'); d.px(11, ty + 4, '#f5d6a8');
  } else if (effect === 'cuoi') {
    const k = frame % 2;
    [[5, 10 - k], [25, 9 + k]].forEach(([x, y]) => { f(x, y, '#ffd23f'); f(x + 1, y + 1, '#ffd23f'); f(x - 1, y + 1, '#ffd23f'); });
  } else if (effect === 'ngu') {
    const z = (x, y) => { f(x, y, INK); f(x + 1, y, INK); f(x + 1, y + 1, INK); f(x, y + 2, INK); f(x + 1, y + 2, INK); };
    z(22, ty - 2 - (frame % 2)); if (frame % 2) z(25, ty - 6);
  }
  drawAcc(opts.acc, f, ty, ey, my);
  if (opts.sick && effect !== 'ngodoc') {
    for (let x = 13; x <= 19; x += 2) f(x, ty + 1, '#5b8def');
    f(18, my, '#ffffff'); f(19, my, '#ffffff'); f(20, my, '#ffffff'); f(21, my, '#e8453c');
  }
  return finish(g, fx, get, set, !opts.noShadow && p.legs === 3);
}

// accessories go on the effect layer so they sit on top of any head shape
function drawAcc(acc, f, ty, ey, my) {
  const row = (x0, x1, y, c) => { for (let x = x0; x <= x1; x++) f(x, y, c); };
  switch (acc) {
  case 'nonla':
    for (let k = 0; k < 4; k++) row(16 - (k * 2) - 1, 16 + (k * 2), ty - 4 + k, k === 3 ? '#b8892f' : '#e8c56a');
    row(7, 25, ty, '#8a6420');
    break;
  case 'kinh':
    [[11, 14], [18, 21]].forEach(([a, b]) => { row(a, b, ey - 1, INK); row(a, b, ey, INK); row(a + 1, b - 1, ey + 1, INK); f(a + 1, ey - 1, '#5bc0f8'); });
    row(15, 17, ey - 1, INK);
    break;
  case 'no':
    [[0, 0], [0, 1], [0, 2], [1, 1], [2, 1], [3, 0], [3, 1], [3, 2]].forEach(([x, y]) => f(19 + x, ty - 2 + y, '#e8453c'));
    f(20, ty - 2, '#ff8fb1');
    break;
  case 'khan':
    row(11, 21, my + 3, '#4064e8'); row(11, 21, my + 4, '#5bc0f8');
    f(20, my + 5, '#4064e8'); f(21, my + 6, '#4064e8'); f(20, my + 6, '#5bc0f8');
    break;
  case 'mu_bao':
    // a domed helmet with a chin strap, so it reads differently from the straw hat
    row(11, 21, ty - 1, '#e8e9ee'); row(12, 20, ty - 2, '#e8e9ee'); row(14, 18, ty - 3, '#ffffff');
    row(10, 22, ty, '#9aa0b5'); f(10, ty + 1, '#9aa0b5'); f(22, ty + 1, '#9aa0b5');
    break;
  case 'vongtay':
    // a plaited grass band around the neck, low and thin
    row(13, 19, my + 4, '#8fbf4d');
    f(12, my + 4, '#6d9636'); f(20, my + 4, '#6d9636'); f(16, my + 5, '#d9e86a');
    break;
  default:
  }
}

function shifted(d, dy) {
  return {...d, ell: (cx, cy, rx, ry, c) => d.ell(cx, cy + dy, rx, ry, c), rect: (a, b, c2, e, c) => d.rect(a, b + dy, c2, e + dy, c), box: (a, b, c2, e, c) => d.box(a, b + dy, c2, e + dy, c)};
}

function drawer(g, set, colors) {
  const d = {
    ...colors,
    shift: 0,
    px: (x, y, c) => set(g, x, y, c),
    rect: (x0, y0, x1, y1, c) => {
      for (let y = y0; y <= y1; y++) for (let x = x0; x <= x1; x++) set(g, x, y + d.shift, c);
    },
    box: (x0, y0, x1, y1, c) => {
      d.rect(x0 + 1, y0, x1 - 1, y1, c);
      d.rect(x0, y0 + 1, x1, y1 - 1, c);
    },
    ell: (cx, cy, rx, ry, c) => {
      for (let y = cy - ry; y <= cy + ry; y++) {
        for (let x = cx - rx; x <= cx + rx; x++) {
          if ((((x - cx) / (rx + 0.5)) ** 2) + (((y - cy) / (ry + 0.5)) ** 2) <= 1) set(g, x, y + d.shift, c);
        }
      }
    },
    tri: (x0, baseY, width, height, c) => {
      for (let k = 0; k < height; k++) {
        const inset = Math.floor((k * width) / (2 * height));
        d.rect(x0 + inset, baseY - k, x0 + width - 1 - inset, baseY - k, c);
      }
    },
  };
  return d;
}

function finish(g, fx, get, set, floating) {
  const out = g.slice();
  for (let y = 0; y < H; y++) {
    for (let x = 0; x < W; x++) {
      if (!g[(y * W) + x] && (get(x - 1, y) || get(x + 1, y) || get(x, y - 1) || get(x, y + 1))) {
        out[(y * W) + x] = OUTLINE;
      }
    }
  }
  if (floating) {
    for (let x = 11; x <= 21; x++) if (!out[((H - 1) * W) + x]) out[((H - 1) * W) + x] = 'rgba(0,0,0,0.18)';
  }
  fx.forEach((c, i) => { if (c) out[i] = c; });
  return out;
}

function paint(ctx, parts, opts = {}) {
  const s = opts.scale || 3;
  const grid = buildGrid(parts, opts);
  ctx.clearRect(0, 0, W * s, H * s);
  for (let i = 0; i < grid.length; i++) {
    if (grid[i]) {
      ctx.fillStyle = grid[i];
      ctx.fillRect((i % W) * s, Math.floor(i / W) * s, s, s);
    }
  }
}

module.exports = {W, H, TOP, COLORS, PARTS, PART_LABELS, randomParts, buildGrid, paint};
