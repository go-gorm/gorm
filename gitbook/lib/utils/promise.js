var Q = require('q');
var _ = require('lodash');

// Reduce an array to a promise
function reduce(arr, iter, base) {
    return _.reduce(arr, function(prev, elem, i) {
        return prev.then(function(val) {
            return iter(val, elem, i);
        });
    }, Q(base));
}

// Transform an array
function serie(arr, iter, base) {
    return reduce(arr, function(before, item, i) {
        return Q(iter(item, i))
        .then(function(r) {
            before.push(r);
            return before;
        });
    }, []);
}

// Iter over an array and return first result (not null)
function some(arr, iter) {
    return _.reduce(arr, function(prev, elem, i) {
        return prev.then(function(val) {
            if (val) return val;

            return iter(elem, i);
        });
    }, Q());
}

// Map an array using an async (promised) iterator
function map(arr, iter) {
    return reduce(arr, function(prev, entry, i) {
        return Q(iter(entry, i))
        .then(function(out) {
            prev.push(out);
            return prev;
        });
    }, []);
}

// Wrap a fucntion in a promise
function wrap(func) {
    return _.wrap(func, function(_func) {
        var args = Array.prototype.slice.call(arguments, 1);
        return Q()
        .then(function() {
            return _func.apply(null, args);
        });
    });
}

module.exports = Q;
module.exports.reduce = reduce;
module.exports.map = map;
module.exports.serie = serie;
module.exports.some = some;
module.exports.wrapfn = wrap;
