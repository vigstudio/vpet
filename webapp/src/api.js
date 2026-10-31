const csrfToken = () => document.cookie.split(';').map((c) => c.trim()).find((c) => c.startsWith('vpet_csrf='))?.slice('vpet_csrf='.length) || '';

export class APIError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

let unauthorizedHandler;
export const setUnauthorizedHandler = (handler) => { unauthorizedHandler = handler; };

async function request(path, body, method = body === undefined ? 'GET' : 'POST') {
  const options = {method, credentials: 'same-origin', headers: {'X-Requested-With': 'XMLHttpRequest'}};
  if (body !== undefined) {
    options.body = JSON.stringify(body);
    Object.assign(options.headers, {'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken()});
  }
  const response = await fetch(path, options);
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    if (response.status === 401) unauthorizedHandler?.();
    throw new APIError(data?.error || `Lỗi ${response.status}`, response.status);
  }
  return data;
}

export const me = () => request('/api/auth/me');
export const login = (username, password) => request('/api/auth/login', {username, password});
export const register = (username, display_name, password, locale) => request('/api/auth/register', {username, display_name, password, locale});
export const claim = (code, username, display_name, password, locale) => request('/api/auth/claim', {code, username, display_name, password, locale});
export const logout = () => request('/api/auth/logout', {});
export const changePassword = (new_password) => request('/api/auth/password', {new_password});
export const updateProfile = (display_name, locale) => request('/api/auth/profile', {display_name, locale});
const game = (path, body) => request(`/api/v1${path}`, body);

export const getState = () => game('/state');
export const searchUsers = async (term) => request(`/api/v1/users/search?q=${encodeURIComponent(term)}`);
export const peekPet = (userId) => game(`/pets/${userId}`);
export const createPet = (name, parts) => game('/pet', {name, parts});
export const useItem = (pet, item) => game('/use', {pet, item});
export const interact = (pet, action) => game('/interact', {pet, action});
export const reskin = (pet, name, parts) => game('/reskin', {pet, name, parts});
export const unequip = (pet) => game('/unequip', {pet});
export const bury = (pet) => game('/bury', {pet});
export const resurrect = (born, died, sacrifice) => game('/resurrect', {born, died, sacrifice});
export const kill = (pet) => game('/kill', {pet});
export const hunt = (pet, map) => game('/hunt', {pet, map});
export const wild = (pet, map) => game('/wild', {pet, map});
export const buy = (item, qty = 1) => game('/buy', {item, qty});
export const gacha = () => game('/gacha', {});
export const equip = (pet, item, level = 0) => game('/equip', {pet, item, level});
export const unequipSlot = (pet, slot) => game('/unequip-slot', {pet, slot});
export const enhance = (pet, slot) => game('/enhance', {pet, slot});
export const restoreEnhanced = (item, level) => game('/enhance/restore', {item, level});
export const skillUp = (pet, skill) => game('/skill-up', {pet, skill});
export const swapSkill = (pet, skill, to) => game('/use', {pet, item: 'sach', skill, to});
export const tower = (pet, stage = 0) => game('/tower', {pet, stage});
export const fuse = (item) => game('/fuse', {item});
export const fuseAll = () => game('/fuse-all', {});
export const equipBest = (pet) => game('/equip-best', {pet});
export const challenge = (to, pet) => game('/duels', {to, pet});
export const duelAction = (id, action, pet = 0) => game(`/duels/${id}/${action}`, {pet});
export const claimAirdrop = () => game('/airdrop', {});
export const setMuted = (muted) => game('/mute', {muted});
export const sellItem = (item, qty, price) => game('/market/sell', {item, qty, price});
export const buyListing = (id) => game('/market/buy', {id});
export const cancelListing = (id) => game('/market/cancel', {id});
export const postBuyOrder = (item, qty, price) => game('/market/want', {item, qty, price});
export const fulfillBuyOrder = (id) => game('/market/fulfill', {id});
export const cancelBuyOrder = (id) => game('/market/cancel-want', {id});
export const renamePet = (pet, name) => game('/rename', {pet, name});
export const sendOffer = (to, give, want) => game('/offers', {to, give, want});
export const offerAction = (id, action) => game(`/offers/${id}/${action}`, {});
export const campaign = (pet, map, stage, difficulty) => game('/campaign', {pet, map, stage, difficulty});
export const allocate = (pet, skill, qty = 1) => game('/build', {pet, skill, qty});
export const bossHit = (pet) => game('/boss/hit', {pet});
export const bossClaim = () => game('/boss/claim', {});
export const party = (action, pet, id = '', map = '') => game('/party', {action, pet, id, map});
export const alliance = (action, payload = {}) => game('/alliance', {action, ...payload});
export const claimQuest = (id) => game('/quest/claim', {id});
export const wearTitle = (pet, id) => game('/title', {pet, id});
export const checkin = () => game('/checkin', {});
export const claimWelcomeGift = () => game('/gift/claim', {});
export const crushGem = (item, qty) => game('/gem/crush', {item, qty});
export const forgeGem = (gem, stage) => game('/gem/forge', {gem, stage});
export const socketGem = (pet, slot, gem) => game('/gem/socket', {pet, slot, gem});
export const unsocketGem = (pet, slot, slot_at) => game('/gem/unsocket', {pet, slot, slot_at});
export const buyCare = (item, qty = 1) => game('/clinic/buy', {item, qty});
export const treatPet = (pet, action) => game('/clinic/treat', {pet, action});
export const boardCare = (pet, days) => game('/clinic/care', {pet, days});

export const worldMessages = () => request('/api/v1/world/messages');
export const sendWorldMessage = (body) => request('/api/v1/world/messages', {body});
export const reportWorldMessage = (message_id) => request('/api/v1/world/report', {message_id});
