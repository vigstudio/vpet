const test = require('node:test');
const assert = require('node:assert/strict');
const {mergeFastState} = require('./state.cjs');

test('fast state preserves unscanned global boards', () => {
    const current = {coin: 10, arena: [{user: 'a'}], market: [{id: 'm'}], buy_orders: [{id: 'b'}]};
    const next = {coin: 11, arena: null, market: null, buy_orders: null};
    assert.deepEqual(mergeFastState(current, next), {...current, coin: 11});
});

test('full state replaces global boards', () => {
    const current = {arena: [{user: 'old'}], market: [{id: 'old'}], buy_orders: [{id: 'old'}]};
    const next = {arena: [], market: [{id: 'new'}], buy_orders: []};
    assert.deepEqual(mergeFastState(current, next), next);
});
