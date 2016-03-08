var _ = require('lodash');
var error = require('../utils/error');

/*
    Return the context for a plugin.
    It tries to keep compatibilities with GitBook v2
*/
function pluginCtx(plugin) {
    var book = plugin.book;
    var ctx = book;

    return ctx;
}

// Call a function "fn" with a context of page similar to the one in GitBook v2
function pageHook(page, fn) {
    var ctx = {
        type: page.type,
        content: page.content,
        path: page.path,
        rawPath: page.rawPath
    };

    // Deprecate sections
    error.deprecateField(ctx, 'sections', [
        { content: ctx.content }
    ], '"sections" property is deprecated, use page.content instead');

    return fn(ctx)
    .then(function(result) {
        if (!result) return undefined;
        if (result.content) {
            return result.content;
        }

        if (result.sections) {
            return _.pluck(result.sections, 'content').join('\n');
        }
    });
}

module.exports = {
    pluginCtx: pluginCtx,
    pageHook: pageHook
};
