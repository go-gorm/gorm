var _ = require('lodash');
var fs = require('fs');
var path = require('path');
var nunjucks = require('nunjucks');

/*
    Nunjucks loader similar to FileSystemLoader, but avoid infinite looping
*/

function isRelative(filename) {
    return (filename.indexOf('./') === 0 || filename.indexOf('../') === 0);
}

var Loader = nunjucks.Loader.extend({
    init: function(searchPaths) {
        this.searchPaths = _.map(searchPaths, path.normalize);
    },

    getSource: function(fullpath) {
        if (!fullpath) return null;

        fullpath = this.resolve(null, fullpath);

        if(!fullpath) {
            return null;
        }

        return {
            src: fs.readFileSync(fullpath, 'utf-8'),
            path: fullpath,
            noCache: true
        };
    },

    // We handle absolute paths ourselves in ".resolve"
    isRelative: function() {
        return true;
    },

    resolve: function(from, to) {
        var searchPaths = this.searchPaths;

        // Relative template like "./test.html"
        if (isRelative(to) && from) {
            return path.resolve(path.dirname(from), to);
        }

        // Determine in which search folder we currently are
        var originalSearchPath = _.chain(this.searchPaths)
            .sortBy(function(s) {
                return -s.length;
            })
            .find(function(basePath) {
                return (from && from.indexOf(basePath) === 0);
            })
            .value();
        var originalFilename = originalSearchPath? path.relative(originalSearchPath, from) : null;

        // If we are including same file from a different search path
        // Slice the search paths to avoid including from previous ones
        if (originalFilename == to) {
            var currentIndex = searchPaths.indexOf(originalSearchPath);
            searchPaths = searchPaths.slice(currentIndex + 1);
        }

        // Absolute template to resolve in root folder
        var resultFolder = _.find(searchPaths, function(basePath) {
            var p = path.resolve(basePath, to);

            return (
                p.indexOf(basePath) === 0
                && fs.existsSync(p)
            );
        });
        if (!resultFolder) return null;
        return path.resolve(resultFolder, to);
    }
});

module.exports = Loader;
