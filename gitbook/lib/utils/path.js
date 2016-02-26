var _ = require('lodash');
var path = require('path');

var error = require('./error');

// Normalize a filename
function normalizePath(filename) {
    return path.normalize(filename);
}

// Return true if file path is inside a folder
function isInRoot(root, filename) {
    filename = path.normalize(filename);
    return (filename.substr(0, root.length) === root);
}

// Resolve paths in a specific folder
// Throw error if file is outside this folder
function resolveInRoot(root) {
    var input, result;

    input = _.chain(arguments)
        .toArray()
        .slice(1)
        .reduce(function(current, p) {
            // Handle path relative to book root ("/README.md")
            if (p[0] == '/' || p[0] == '\\') return p.slice(1);

            return current? path.join(current, p) : path.normalize(p);
        }, '')
        .value();

    result = path.resolve(root, input);

    if (!isInRoot(root, result)) {
        throw new error.FileOutOfScopeError({
            filename: result,
            root: root
        });
    }

    return result;
}

// Chnage extension
function setExtension(filename, ext) {
    return path.join(
        path.dirname(filename),
        path.basename(filename, path.extname(filename)) + ext
    );
}

module.exports = {
    isInRoot: isInRoot,
    resolveInRoot: resolveInRoot,
    normalize: normalizePath,
    setExtension: setExtension
};
