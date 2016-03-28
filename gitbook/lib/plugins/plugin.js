var _ = require('lodash');
var path = require('path');
var url = require('url');
var resolve = require('resolve');
var mergeDefaults = require('merge-defaults');
var jsonschema = require('jsonschema');
var jsonSchemaDefaults = require('json-schema-defaults');

var Promise = require('../utils/promise');
var error = require('../utils/error');
var gitbook = require('../gitbook');
var registry = require('./registry');
var compatibility = require('./compatibility');

var HOOKS = [
    'init', 'finish', 'finish:before', 'config', 'page', 'page:before'
];

var RESOURCES = ['js', 'css'];

// Return true if an error is a "module not found"
// Wait on https://github.com/substack/node-resolve/pull/81 to be merged
function isModuleNotFound(err) {
    return err.message.indexOf('Cannot find module') >= 0;
}

function BookPlugin(book, pluginId, pluginFolder) {
    this.book = book;
    this.log = this.book.log.prefix(pluginId);


    this.id = pluginId;
    this.npmId = registry.npmId(pluginId);
    this.root = pluginFolder;

    this.packageInfos = undefined;
    this.content = undefined;

    // Cache for resources
    this._resources = {};

    _.bindAll(this);
}

// Return true if plugin has been loaded correctly
BookPlugin.prototype.isLoaded = function() {
    return Boolean(this.packageInfos && this.content);
};

// Bind a function to the plugin's context
BookPlugin.prototype.bind = function(fn) {
    return fn.bind(compatibility.pluginCtx(this));
};

// Load this plugin from its root folder
BookPlugin.prototype.load = function(folder) {
    var that = this;

    if (this.isLoaded()) {
        return Promise.reject(new Error('Plugin "' + this.id + '" is already loaded'));
    }

    // Try loading plugins from different location
    var p = Promise()
    .then(function() {
        // Locate plugin and load pacjage.json
        try {
            var res = resolve.sync('./package.json', { basedir: that.root });

            that.root = path.dirname(res);
            that.packageInfos = require(res);
        } catch (err) {
            if (!isModuleNotFound(err)) throw err;

            that.packageInfos = undefined;
            that.content = undefined;

            return;
        }

        // Load plugin JS content
        try {
            that.content = require(that.root);
        } catch(err) {
            // It's no big deal if the plugin doesn't have an "index.js"
            // (For example: themes)
            if (isModuleNotFound(err)) {
                that.content = {};
            } else {
                throw new error.PluginError(err, {
                    plugin: that.id
                });
            }
        }
    })

    .then(that.validate)

    // Validate the configuration and update it
    .then(function() {
        var config = that.book.config.get(that.getConfigKey(), {});
        return that.validateConfig(config);
    })
    .then(function(config) {
        that.book.config.set(that.getConfigKey(), config);
    });

    this.log.info('loading plugin "' + this.id + '"... ');
    return this.log.info.promise(p);
};

// Verify the definition of a plugin
// Also verify that the plugin accepts the current gitbook version
// This method throws erros if plugin is invalid
BookPlugin.prototype.validate = function() {
    var isValid = (
        this.isLoaded() &&
        this.packageInfos &&
        this.packageInfos.name &&
        this.packageInfos.engines &&
        this.packageInfos.engines.gitbook
    );

    if (!isValid) {
        throw new Error('Error loading plugin "' + this.id + '" at "' + this.root + '"');
    }

    if (!gitbook.satisfies(this.packageInfos.engines.gitbook)) {
        throw new Error('GitBook doesn\'t satisfy the requirements of this plugin: '+this.packageInfos.engines.gitbook);
    }
};

// Normalize, validate configuration for this plugin using its schema
// Throw an error when shcema is not respected
BookPlugin.prototype.validateConfig = function(config) {
    var that = this;

    return Promise()
    .then(function() {
        var schema = that.packageInfos.gitbook || {};
        if (!schema) return config;

        // Normalize schema
        schema.id = '/'+that.getConfigKey();
        schema.type = 'object';

        // Validate and throw if invalid
        var v = new jsonschema.Validator();
        var result = v.validate(config, schema, {
            propertyName: that.getConfigKey()
        });

        // Throw error
        if (result.errors.length > 0) {
            throw new error.ConfigurationError(new Error(result.errors[0].stack));
        }

        // Insert default values
        var defaults = jsonSchemaDefaults(schema);
        return mergeDefaults(config, defaults);
    });
};

// Return key for configuration
BookPlugin.prototype.getConfigKey = function() {
    return 'pluginsConfig.'+this.id;
};

// Call a hook and returns its result
BookPlugin.prototype.hook = function(name, input) {
    var that = this;
    var hookFunc = this.content.hooks? this.content.hooks[name] : null;
    input = input || {};

    if (!hookFunc) return Promise(input);

    this.book.log.debug.ln('call hook "' + name + '" for plugin "' + this.id + '"');
    if (!_.contains(HOOKS, name)) {
        this.book.log.warn.ln('hook "'+name+'" used by plugin "'+this.name+'" is deprecated, and will be removed in the coming versions');
    }

    return Promise()
    .then(function() {
        return that.bind(hookFunc)(input);
    });
};

// Return resources without normalization
BookPlugin.prototype._getResources = function(base) {
    var that = this;

    return Promise()
    .then(function() {
        if (that._resources[base]) return that._resources[base];

        var book = that.content[base];

        // Compatibility with version 1.x.x
        if (base == 'website') book = book || that.content.book;

        // Nothing specified, fallback to default
        if (!book) {
            return Promise({});
        }

        // Dynamic function
        if(typeof book === 'function') {
            // Call giving it the context of our book
            return that.bind(book)();
        }

        // Plain data object
        return book;
    })

    .then(function(resources) {
        that._resources[base] = resources;
        return _.cloneDeep(resources);
    });
};

// Normalize a specific resource
BookPlugin.prototype.normalizeResource = function(resource) {
    // Parse the resource path
    var parsed = url.parse(resource);

    // This is a remote resource
    // so we will simply link to using it's URL
    if (parsed.protocol) {
        return {
            'url': resource
        };
    }

    // This will be copied over from disk
    // and shipped with the book's build
    return { 'path': this.npmId+'/'+resource };
};


// Normalize resources and return them
BookPlugin.prototype.getResources = function(base) {
    var that = this;

    return this._getResources(base)
    .then(function(resources) {
        _.each(RESOURCES, function(resourceType) {
            resources[resourceType] = _.map(resources[resourceType] || [], that.normalizeResource);
        });

        return resources;
    });
};

// Normalize filters and return them
BookPlugin.prototype.getFilters = function() {
    var that = this;

    return _.mapValues(this.content.filters || {}, function(fn, filter) {
        return function() {
            var ctx = _.extend(compatibility.pluginCtx(that), this);

            return fn.apply(ctx, arguments);
        };
    });
};

// Normalize blocks and return them
BookPlugin.prototype.getBlocks = function() {
    var that = this;

    return _.mapValues(this.content.blocks || {}, function(block, blockName) {
        block = _.isFunction(block)? { process: block } : block;

        var fn = block.process;
        block.process = function() {
            var ctx = _.extend(compatibility.pluginCtx(that), this);

            return fn.apply(ctx, arguments);
        };

        return block;
    });
};

module.exports = BookPlugin;
module.exports.RESOURCES = RESOURCES;

