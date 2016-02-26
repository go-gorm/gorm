var _ = require('lodash');
var semver = require('semver');

var gitbook = require('../gitbook');
var Promise = require('../utils/promise');
var validator = require('./validator');
var plugins = require('./plugins');

// Config files to tested (sorted)
var CONFIG_FILES = [
    'book.js',
    'book.json'
];

/*
Config is an interface for the book's configuration stored in "book.json" (or "book.js")
*/

function Config(book, baseConfig) {
    this.book = book;
    this.fs = book.fs;
    this.log = book.log;
    this.path = '';

    this.baseConfig = baseConfig || {};
    this.replace({});
}

// Load configuration of the book
// and verify that the configuration is satisfying
Config.prototype.load = function() {
    var that = this;
    var isLanguageBook = this.book.isLanguageBook();

    // Try all potential configuration file
    return Promise.some(CONFIG_FILES, function(filename) {
        that.log.debug.ln('try loading configuration from', filename);

        return that.fs.loadAsObject(that.book.resolve(filename))
        .then(function(_config) {
            that.log.debug.ln('configuration loaded from', filename);

            that.path = filename;
            return that.replace(_config);
        })
        .fail(function(err) {
            if (err.code != 'MODULE_NOT_FOUND') throw(err);
            else return Promise(false);
        });
    })
    .then(function() {
        if (!isLanguageBook) {
            if (!gitbook.satisfies(that.options.gitbook)) {
                throw new Error('GitBook version doesn\'t satisfy version required by the book: '+that.options.gitbook);
            }
            if (that.options.gitbook != '*' && !semver.satisfies(semver.inc(gitbook.version, 'patch'), that.options.gitbook)) {
                that.log.warn.ln('gitbook version specified in your book.json might be too strict for future patches, \''+(_.first(gitbook.version.split('.'))+'.x.x')+'\' is more adequate');
            }

            that.options.plugins = plugins.toList(that.options.plugins);
        } else {
            // Multilingual book should inherits the plugins list from parent
            that.options.plugins = that.book.parent.config.get('plugins');
        }

        that.options.gitbook = gitbook.version;
    });
};

// Replace the whole configuration
Config.prototype.replace = function(options) {
    var that = this;

    // Extend base config
    options = _.defaults(_.cloneDeep(options), this.baseConfig);

    // Validate the config
    this.options = validator.validate(options);

    // options.input == book.root
    Object.defineProperty(this.options, 'input', {
        get: function () {
            return that.book.root;
        }
    });

    // options.originalInput == book.parent.root
    Object.defineProperty(this.options, 'originalInput', {
        get: function () {
            return that.book.parent? that.book.parent.root : undefined;
        }
    });
};

// Return true if book has a configuration file
Config.prototype.exists = function() {
    return Boolean(this.path);
};

// Return path to a structure file
// Strip the extension by default
Config.prototype.getStructure = function(name, dontStripExt) {
    var filename = this.options.structure[name];
    if (dontStripExt) return filename;

    filename = filename.split('.').slice(0, -1).join('.');
    return filename;
};

// Return a configuration using a key and a default value
Config.prototype.get = function(key, def) {
    return _.get(this.options, key, def);
};

// Update a configuration
Config.prototype.set = function(key, value) {
    return _.set(this.options, key, value);
};

// Return a dump of the configuration
Config.prototype.dump = function() {
    return _.cloneDeep(this.options);
};

// Return templating context
Config.prototype.getContext = function() {
    return {
        config: this.book.config.dump()
    };
};

module.exports = Config;
