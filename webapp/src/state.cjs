function mergeFastState(current, next) {
    if (!next) return next;
    return {
        ...current,
        ...next,
        market: next.market ?? current?.market,
        buy_orders: next.buy_orders ?? current?.buy_orders,
        arena: next.arena ?? current?.arena,
    };
}

module.exports = {mergeFastState};
