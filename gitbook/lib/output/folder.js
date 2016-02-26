var _ = require('lodash');
var util = require('util');
var path = require('path');

var Output = require('./base');
var fs = require('../utils/fs');
var pathUtil = require('../utils/path');
var Promise = require('../utils/promise');

/*
This output requires the native fs module to output
book as a directory (mapping assets and pages)
*/

module.exports = function folderOutput(Base) {
    Base = Base || Output;

    function FolderOutput() {
        Base.apply(this, arguments);

        this.opts.root = path.resolve(this.opts.root || this.book.resolve('_book'));
    }
    util.inherits(FolderOutput, Base);

    // Copy an asset file (non-parsable), ex: images, etc
    FolderOutput.prototype.onAsset = function(filename) {
        return this.copyFile(
            this.book.resolve(filename),
            filename
        );
    };

    // Prepare the generation by creating the output folder
    FolderOutput.prototype.prepare = function() {
        var that = this;

        return Promise()
        .then(function() {
            return FolderOutput.super_.prototype.prepare.apply(that);
        })

        // Cleanup output folder
        .then(function() {
            that.log.debug.ln('removing previous output directory');
            return fs.rmDir(that.root())
            .fail(function() {
                return Promise();
            });
        })

        // Create output folder
        .then(function() {
            that.log.debug.ln('creating output directory');
            return fs.mkdirp(that.root());
        })

        // Add output folder to ignored files
        .then(function() {
            that.ignore.addPattern([
                path.relative(that.book.root, that.root())
            ]);
        });
    };

    // Prepare output for a language book
    FolderOutput.prototype.onLanguageBook = function(book) {
        return new this.constructor(book, _.extend({}, this.opts, {

            // Language output should be output in sub-directory of output
            root: path.resolve(this.root(), book.language)
        }), this);
    };

    // ----- Utility methods -----

    // Return path to the root folder
    FolderOutput.prototype.root = function() {
        return this.opts.root;
    };

    // Resolve a file in the output directory
    FolderOutput.prototype.resolve = function(filename) {
        return pathUtil.resolveInRoot.apply(null, [this.root()].concat(_.toArray(arguments)));
    };

    // Copy a file to the output
    FolderOutput.prototype.copyFile = function(from, to) {
        var that = this;

        return Promise()
        .then(function() {
            to = that.resolve(to);
            var folder = path.dirname(to);

            // Ensure folder exists
            return fs.mkdirp(folder);
        })
        .then(function() {
            return fs.copy(from, to);
        });
    };

    // Write a file/buffer to the output folder
    FolderOutput.prototype.writeFile = function(filename, buf) {
        var that = this;

        return Promise()
        .then(function() {
            filename = that.resolve(filename);
            var folder = path.dirname(filename);

            // Ensure folder exists
            return fs.mkdirp(folder);
        })

        // Write the file
        .then(function() {
            return fs.writeFile(filename, buf);
        });
    };

    // Return true if a file exists in the output folder
    FolderOutput.prototype.hasFile = function(filename) {
        var that = this;

        return Promise()
        .then(function() {
            return fs.exists(that.resolve(filename));
        });
    };

    // Create a new unique file
    // Returns its filename
    FolderOutput.prototype.createNewFile = function(base, filename) {
        var that = this;

        if (!filename) {
            filename = path.basename(filename);
            base = path.dirname(base);
        }

        return fs.uniqueFilename(this.resolve(base), filename)
            .then(function(out) {
                out = path.join(base, out);

                return fs.ensure(that.resolve(out))
                    .thenResolve(out);
            });
    };

    return FolderOutput;
};
