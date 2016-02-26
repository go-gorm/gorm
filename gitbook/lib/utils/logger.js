var _ = require('lodash');
var util = require('util');
var color = require('bash-color');

var LEVELS = {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
    DISABLED: 10
};

var COLORS = {
    DEBUG: color.purple,
    INFO: color.cyan,
    WARN: color.yellow,
    ERROR: color.red
};

function Logger(write, logLevel, prefix) {
    if (!(this instanceof Logger)) return new Logger(write, logLevel);

    this._write = write || function(msg) { process.stdout.write(msg); };
    this.lastChar = '\n';

    // Define log level
    this.setLevel(logLevel);

    _.bindAll(this);

    // Create easy-to-use method like "logger.debug.ln('....')"
    _.each(_.omit(LEVELS, 'DISABLED'), function(level, levelKey) {
        levelKey = levelKey.toLowerCase();

        this[levelKey] = _.partial(this.log, level);
        this[levelKey].ln = _.partial(this.logLn, level);
        this[levelKey].ok = _.partial(this.ok, level);
        this[levelKey].fail = _.partial(this.fail, level);
        this[levelKey].promise = _.partial(this.promise, level);
    }, this);
}

// Create a new logger prefixed from this logger
Logger.prototype.prefix = function(prefix) {
    return (new Logger(this._write, this.logLevel, prefix));
};

// Change minimum level
Logger.prototype.setLevel = function(logLevel) {
    if (_.isString(logLevel)) logLevel = LEVELS[logLevel.toUpperCase()];
    this.logLevel = logLevel;
};

// Print a simple string
Logger.prototype.write = function(msg) {
    msg = msg.toString();
    this.lastChar = _.last(msg);
    return this._write(msg);
};

// Format a string using the first argument as a printf-like format.
Logger.prototype.format = function() {
    return util.format.apply(util, arguments);
};

// Print a line
Logger.prototype.writeLn = function(msg) {
    return this.write((msg || '')+'\n');
};

// Log/Print a message if level is allowed
Logger.prototype.log = function(level) {
    if (level < this.logLevel) return;

    var levelKey = _.findKey(LEVELS, function(v) { return v == level; });
    var args = Array.prototype.slice.apply(arguments, [1]);
    var msg = this.format.apply(this, args);

    if (this.lastChar == '\n') {
        msg = COLORS[levelKey](levelKey.toLowerCase()+':')+' '+msg;
    }

    return this.write(msg);
};

// Log/Print a line if level is allowed
Logger.prototype.logLn = function() {
    if (this.lastChar != '\n') this.write('\n');

    var args = Array.prototype.slice.apply(arguments);
    args.push('\n');
    return this.log.apply(this, args);
};

// Log a confirmation [OK]
Logger.prototype.ok = function(level) {
    var args = Array.prototype.slice.apply(arguments, [1]);
    var msg = this.format.apply(this, args);
    if (arguments.length > 1) {
        this.logLn(level, color.green('>> ') + msg.trim().replace(/\n/g, color.green('\n>> ')));
    } else {
        this.log(level, color.green('OK'), '\n');
    }
};

// Log a "FAIL"
Logger.prototype.fail = function(level) {
    return this.log(level, color.red('ERROR') + '\n');
};

// Log state of a promise
Logger.prototype.promise = function(level, p) {
    var that = this;

    return p.
    then(function(st) {
        that.ok(level);
        return st;
    }, function(err) {
        that.fail(level);
        throw err;
    });
};

Logger.LEVELS = LEVELS;
Logger.COLORS = COLORS;

module.exports =  Logger;
