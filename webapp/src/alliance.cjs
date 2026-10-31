const rankRows = (rows = []) => [...rows].sort((a, b) => a.rank - b.rank);

const influenceFor = (land, alliance) => Number(land?.influence?.[alliance] || 0);

const warCountdown = (ends, now) => {
  const seconds = Math.max(0, Math.ceil((Number(ends || 0) - Number(now || 0)) / 1000));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${String(minutes).padStart(2, '0')}m`;
};

module.exports = {influenceFor, rankRows, warCountdown};
