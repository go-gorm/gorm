var _ = require('lodash');

module.exports = {
    // Return non-parsed html
    // since blocks are by default non-parsable, a simple identity method works fine
    html: _.identity,

    // Highlight a code block
    // This block can be replaced by plugins
    code: function(blk) {
        return {
            html: false,
            body: blk.body
        };
    },

    // Render some markdown to HTML
    markdown: function(blk) {
        return this.book.renderInline('markdown', blk.body)
        .then(function(out) {
            return { body: out };
        });
    },
    asciidoc: function(blk) {
        return this.book.renderInline('asciidoc', blk.body)
        .then(function(out) {
            return { body: out };
        });
    },
    markup: function(blk) {
        return this.book.renderInline(this.ctx.file.type, blk.body)
        .then(function(out) {
            return { body: out };
        });
    }
};
