const gainedItems = (before = {}, after = {}) => Object.keys(after).filter((id) => (after[id] || 0) > (before[id] || 0));

module.exports = {gainedItems};
