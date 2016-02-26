var _ = require('lodash');
var path = require('path');

var Promise = require('../utils/promise');
var fs = require('../utils/fs');
var BookPlugin = require('./plugin');
var registry = require('./registry');
var pluginsConfig = require('../config/plugins');

/*
PluginsManager is an interface to work with multiple plugins at once:
- Extract assets from plugins
- Call hooks for all plugins, etc
*/

function PluginsManager(book) {
    this.book = book;
    this.log = this.book.log;
    this.plugins = [];

    _.bindAll(this);
}

// Return count of plugins loaded
PluginsManager.prototype.count = function() {
    return _.size(this.plugins);
};

// Returns a plugin by its name
PluginsManager.prototype.get = function(name) {
    return _.find(this.plugins, {
        id: name
    });
};

// Load a plugin, or a list of plugins
PluginsManager.prototype.load = function(name) {
    var that = this;

    if (_.isArray(name)) {
        return Promise.serie(name, function(_name) {
            return that.load(_name);
        });
    }

    return Promise()

    // Initiate and load the plugin
    .then(function() {
        var plugin;

        if (!_.isString(name)) plugin = name;
        else plugin = new BookPlugin(that.book, name);

        if (that.get(plugin.id)) {
            throw new Error('Plugin "'+plugin.id+'" is already loaded');
        }


        if (plugin.isLoaded()) return plugin;
        else return plugin.load()
            .thenResolve(plugin);
    })

    // Setup the plugin
    .then(this._setup);
};

// Load all plugins from the book's configuration
PluginsManager.prototype.loadAll = function() {
    var plugins = _.pluck(this.book.config.get('plugins'), 'name');

    this.log.info.ln('loading', plugins.length, 'plugins');
    return this.load(plugins);
};

// Setup a plugin
// Register its filter, blocks, etc
PluginsManager.prototype._setup = function(plugin) {
    this.plugins.push(plugin);
};

// Install all plugins for the book
PluginsManager.prototype.install = function() {
    var that = this;
    var plugins = _.filter(this.book.config.get('plugins'), function(plugin) {
        return !pluginsConfig.isDefaultPlugin(plugin.name);
    });

    if (plugins.length == 0) {
        this.log.info.ln('nothing to install!');
        return Promise(0);
    }

    this.log.info.ln('installing', plugins.length, 'plugins');

    return Promise.serie(plugins, function(plugin) {
        return registry.install(that.book, plugin.name, plugin.version);
    })
    .thenResolve(plugins.length);
};

// Call a hook on all plugins to transform an input
PluginsManager.prototype.hook = function(name, input) {
    return Promise.reduce(this.plugins, function(current, plugin) {
        return plugin.hook(name, current);
    }, input);
};

// Extract all resources for a namespace
PluginsManager.prototype.getResources = function(namespace) {
    return Promise.reduce(this.plugins, function(out, plugin) {
        return plugin.getResources(namespace)
        .then(function(pluginResources) {
            _.each(BookPlugin.RESOURCES, function(resourceType) {
                out[resourceType] = (out[resourceType] || []).concat(pluginResources[resourceType] || []);
            });

            return out;
        });
    }, {});
};

// Copy all resources for a plugin
PluginsManager.prototype.copyResources = function(namespace, outputRoot) {
    return Promise.serie(this.plugins, function(plugin) {
        return plugin.getResources(namespace)
        .then(function(resources) {
            if (!resources.assets) return;

            var input = path.resolve(plugin.root, resources.assets);
            var output = path.resolve(outputRoot, plugin.npmId);

            return fs.copyDir(input, output);
        });
    });
};

// Get all filters and blocks
PluginsManager.prototype.getFilters = function() {
    return _.reduce(this.plugins, function(out, plugin) {
        var filters = plugin.getFilters();

        return _.extend(out, filters);
    }, {});
};
PluginsManager.prototype.getBlocks = function() {
    return _.reduce(this.plugins, function(out, plugin) {
        var blocks = plugin.getBlocks();

        return _.extend(out, blocks);
    }, {});
};

module.exports = PluginsManager;
