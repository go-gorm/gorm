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
    }
};
