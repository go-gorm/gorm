var semver = require('semver');
var pkg = require('../package.json');

var VERSION = pkg.version;
var VERSION_STABLE = VERSION.replace(/\-(\S+)/g, '');

var START_TIME = new Date();

// Verify that this gitbook version satisfies a requirement
// We can't directly use samver.satisfies since it will break all plugins when gitbook version is a prerelease (beta, alpha)
function satisfies(condition) {
    // Test with real version
    if (semver.satisfies(VERSION, condition)) return true;

    // Test with future stable release
    return semver.satisfies(VERSION_STABLE, condition);
}

// Return templating/json context for gitbook itself
function getContext() {
    return {
        gitbook: {
            version: pkg.version,
            time: START_TIME
        }
    };
}

module.exports = {
    version: pkg.version,
    satisfies: satisfies,
    getContext: getContext
};
