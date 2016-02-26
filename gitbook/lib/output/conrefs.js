var path = require('path');
var util = require('util');

var folderOutput = require('./folder');
var Git = require('../utils/git');
var fs = require('../utils/fs');
var pathUtil = require('../utils/path');
var location = require('../utils/location');

/*
Mixin for output to resolve git conrefs
*/

module.exports = function conrefsLoader(Base) {
    Base = folderOutput(Base);

    function ConrefsLoader() {
        Base.apply(this, arguments);

        this.git = new Git();
    }
    util.inherits(ConrefsLoader, Base);

    // Read a template by its source URL
    ConrefsLoader.prototype.onGetTemplate = function(sourceURL) {
        var that = this;

        return this.git.resolve(sourceURL)
        .then(function(filepath) {
            // Is local file
            if (!filepath) {
                filepath = that.book.resolve(sourceURL);
            } else {
                that.book.log.debug.ln('resolve from git', sourceURL, 'to', filepath);
            }

            //  Read file from absolute path
            return fs.readFile(filepath)
            .then(function(source) {
                return {
                    src: source.toString('utf8'),
                    path: filepath
                };
            });
        });
    };

    // Generate a source URL for a template
    ConrefsLoader.prototype.onResolveTemplate = function(from, to) {
        // If origin is in the book, we enforce result file to be in the book
        if (this.book.isInBook(from)) {
            var href = location.toAbsolute(to, path.dirname(from), '');
            return this.book.resolve(href);
        }

        // If origin is in a git repository, we resolve file in the git repository
        var gitRoot = this.git.resolveRoot(from);
        if (gitRoot) {
            return pathUtil.resolveInRoot(gitRoot, to);
        }

        // If origin is not in the book (include from a git content ref)
        return path.resolve(path.dirname(from), to);
    };

    return ConrefsLoader;
};
