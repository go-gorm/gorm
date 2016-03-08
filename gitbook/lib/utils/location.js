var url = require('url');
var path = require('path');

// Is the url an external url
function isExternal(href) {
    try {
        return Boolean(url.parse(href).protocol);
    } catch(err) {
        return false;
    }
}

// Inverse of isExternal
function isRelative(href) {
    return !isExternal(href);
}

// Return true if the link is an achor
function isAnchor(href) {
    try {
        var parsed = url.parse(href);
        return !!(!parsed.protocol && !parsed.path && parsed.hash);
    } catch(err) {
        return false;
    }
}

// Normalize a path to be a link
function normalize(s) {
    return path.normalize(s).replace(/\\/g, '/');
}

// Convert relative to absolute path
// dir: directory parent of the file currently in rendering process
// outdir: directory parent from the html output
function toAbsolute(_href, dir, outdir) {
    if (isExternal(_href)) return _href;
    outdir = outdir == undefined? dir : outdir;

    _href = normalize(_href);
    dir = normalize(dir);
    outdir = normalize(outdir);

    // Path "_href" inside the base folder
    var hrefInRoot = path.normalize(path.join(dir, _href));
    if (_href[0] == '/') hrefInRoot = path.normalize(_href.slice(1));

    // Make it relative to output
    _href = path.relative(outdir, hrefInRoot);

    // Normalize windows paths
    _href = normalize(_href);

    return _href;
}

// Convert an absolute path to a relative path for a specific folder (dir)
// ('test/', 'hello.md') -> '../hello.md'
function relative(dir, file) {
    return normalize(path.relative(dir, file));
}

module.exports = {
    isExternal: isExternal,
    isRelative: isRelative,
    isAnchor: isAnchor,
    normalize: normalize,
    toAbsolute: toAbsolute,
    relative: relative
};
