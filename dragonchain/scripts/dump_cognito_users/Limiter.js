// 09/2018 // 
// Queues - limits api calls //

const Bottleneck = require('bottleneck');

class Limiter {
    constructor(maxConcurrent=null, minTime=0, highWater=null, strategy=null, penalty=null) {
        this.concurrency = maxConcurrent;
        this.minTime = minTime;
        this.highWater = highWater;
        this.penalty = penalty;
        this.config = { maxConcurrent, minTime, highWater };
        this.strategies = {
            leak: Bottleneck.strategy.LEAK,
            overflow_priority: Bottleneck.strategy.OVERFLOW_PRIORITY,
            overflow: Bottleneck.strategy.OVERFLOW,
            block: Bottleneck.strategy.BLOCK,
        };
        if (strategy) {
            this.config.strategy = this.strategies[strategy];
        }
        this.limiter = new Bottleneck(this.config);
    }
};

module.exports = Limiter;