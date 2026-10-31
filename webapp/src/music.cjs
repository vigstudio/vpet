const MUSIC_KEY = 'vpet.music';

function readMusic(storage) {
  try {
    return storage.getItem(MUSIC_KEY) === 'on';
  } catch (_) {
    return false;
  }
}

function writeMusic(storage, enabled) {
  try {
    storage.setItem(MUSIC_KEY, enabled ? 'on' : 'off');
  } catch (_) {
    // Private browsing may deny storage; music still works for this session.
  }
}

module.exports = {MUSIC_KEY, readMusic, writeMusic};
