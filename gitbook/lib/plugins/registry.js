var npm = require('npm');
var npmi = require('npmi');
var semver = require('semver');
var _ = require('lodash');

var Promise = require('../utils/promise');
var gitbook = require('../gitbook');

var PLUGIN_PREFIX = 'gitbook-plugin-';

// Return an absolute name for the plugin (the one on NPM)
function npmId(name) {
    if (name.indexOf(PLUGIN_PREFIX) === 0) return name;
    return [PLUGIN_PREFIX, name].join('');
}

// Return a plugin ID 9the one on GitBook
function pluginId(name) {
    return name.replace(PLUGIN_PREFIX, '');
}

// Validate an NPM plugin ID
function validateId(name) {
    return name.indexOf(PLUGIN_PREFIX) === 0;
}

// Initialize NPM for operations
var initNPM = _.memoize(function() {
    return Promise.nfcall(npm.load, {
        silent: true,
        loglevel: 'silent'
    });
});

// Link a plugin for use in a specific book
function linkPlugin(book, pluginPath) {
    book.log('linking', pluginPath);
}

// Resolve the latest version for a plugin
function resolveVersion(plugin) {
    var npnName = npmId(plugin);

    return initNPM()
    .then(function() {
        return Promise.nfcall(npm.commands.view, [npnName+'@*', 'engines'], true);
    })
    .then(function(versions) {
        return _.chain(versions)
            .pairs()
            .map(function(v) {
                return {
                    version: v[0],
                    gitbook: (v[1].engines || {}).gitbook
                };
            })
            .filter(function(v) {
                return v.gitbook && gitbook.satisfies(v.gitbook);
            })
            .sort(function(v1, v2) {
                return semver.lt(v1.version, v2.version)? 1 : -1;
            })
            .pluck('version')
            .first()
            .value();
    });
}


// Install a plugin in a book
function installPlugin(book, plugin, version) {
    book.log.info.ln('installing plugin', plugin);

    var npnName = npmId(plugin);

    return Promise()
    .then(function() {
        if (version) return version;

        book.log.info.ln('No version specified, resolve plugin "' + plugin + '"');
        return resolveVersion(plugin);
    })

    // Install the plugin with the resolved version
    .then(function(version) {
        if (!version) {
            throw new Error('Found no satisfactory version for plugin "' + plugin + '"');
        }

        book.log.info.ln('install plugin "' + plugin +'" from npm ('+npnName+') with version', version);
        return Promise.nfcall(npmi, {
            'name': npnName,
            'version': version,
            'path': book.root,
            'npmLoad': {
                'loglevel': 'silent',
                'loaded': true,
                'prefix': book.root
            }
        });
    })
    .then(function() {
        book.log.info.ok('plugin "' + plugin + '" installed with success');
    });
}

module.exports = {
    npmId: npmId,
    pluginId: pluginId,
    validateId: validateId,

    resolve: resolveVersion,
    link: linkPlugin,
    install: installPlugin
};
