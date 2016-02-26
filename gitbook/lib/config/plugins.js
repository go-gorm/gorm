var _ = require('lodash');

// Default plugins added to each books
var DEFAULT_PLUGINS = ['highlight', 'search', 'sharing', 'fontsettings', 'theme-default'];

// Return true if a plugin is a default plugin
function isDefaultPlugin(name, version) {
    return _.contains(DEFAULT_PLUGINS, name);
}

// Normalize a list of plugins to use
function normalizePluginsList(plugins) {
    // Normalize list to an array
    plugins = _.isString(plugins) ? plugins.split(',') : (plugins || []);

    // Remove empty parts
    plugins = _.compact(plugins);

    // Divide as {name, version} to handle format like 'myplugin@1.0.0'
    plugins = _.map(plugins, function(plugin) {
        if (plugin.name) return plugin;

        var parts = plugin.split('@');
        var name = parts[0];
        var version = parts[1];
        return {
            'name': name,
            'version': version // optional
        };
    });

    // List plugins to remove
    var toremove = _.chain(plugins)
    .filter(function(plugin) {
        return plugin.name.length > 0 && plugin.name[0] == '-';
    })
    .map(function(plugin) {
        return plugin.name.slice(1);
    })
    .value();

    // Merge with defaults
    _.each(DEFAULT_PLUGINS, function(plugin) {
        if (_.find(plugins, { name: plugin })) {
            return;
        }

        plugins.push({
            'name': plugin
        });
    });
    // Remove plugin that start with '-'
    plugins = _.filter(plugins, function(plugin) {
        return !_.contains(toremove, plugin.name) && !(plugin.name.length > 0 && plugin.name[0] == '-');
    });

    // Remove duplicates
    plugins = _.uniq(plugins, 'name');

    return plugins;
}

module.exports = {
    isDefaultPlugin: isDefaultPlugin,
    toList: normalizePluginsList
};

