var _ = require('lodash');
var path = require('path');
var chokidar = require('chokidar');

var Promise = require('../utils/promise');
var parsers = require('../parsers');

// Watch a folder and resolve promise once a file is modified
function watch(dir) {
    var d = Promise.defer();
    dir = path.resolve(dir);

    var toWatch = [
        'book.json', 'book.js'
    ];

    // Watch all parsable files
    _.each(parsers.extensions, function(ext) {
        toWatch.push('**/*'+ext);
    });

    var watcher = chokidar.watch(toWatch, {
        cwd: dir,
        ignored: '_book/**',
        ignoreInitial: true
    });

    watcher.once('all', function(e, filepath) {
        watcher.close();

        d.resolve(filepath);
    });
    watcher.once('error', function(err) {
        watcher.close();

        d.reject(err);
    });

    return d.promise;
}

module.exports = watch;
