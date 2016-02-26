var nunjucks = require('nunjucks');
var location = require('../utils/location');

/*
Simple nunjucks loader which is passing the reponsability to the Output
*/

var Loader = nunjucks.Loader.extend({
    async: true,

    init: function(engine, opts) {
        this.engine = engine;
        this.output = engine.output;
    },

    getSource: function(sourceURL, callback) {
        var that = this;

        this.output.onGetTemplate(sourceURL)
        .then(function(out) {
            // We disable cache since content is modified (shortcuts, ...)
            out.noCache = true;

            // Transform template before runnign it
            out.source = that.engine.interpolate(out.path, out.source);

            return out;
        })
        .nodeify(callback);
    },

    resolve: function(from, to) {
        return this.output.onResolveTemplate(from, to);
    },

    // Handle all files as relative, so that nunjucks pass responsability to 'resolve'
    isRelative: function(filename) {
        return location.isRelative(filename);
    }
});

module.exports = Loader;
