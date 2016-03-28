var _ = require('lodash');
var path = require('path');
var util = require('util');
var nunjucks = require('nunjucks');
var I18n = require('i18n-t');

var Promise = require('../utils/promise');
var location = require('../utils/location');
var fs = require('../utils/fs');
var defaultFilters = require('../template/filters');
var FSLoader = require('../template/fs-loader');
var conrefsLoader = require('./conrefs');
var Output = require('./base');


// Directory for a theme with the templates
function templatesPath(dir) {
    return path.join(dir, '_layouts');
}

function _WebsiteOutput() {
    Output.apply(this, arguments);

    // Nunjucks environment
    this.env;

    // Plugin instance for the main theme
    this.theme;

    // Plugin instance for the default theme
    this.defaultTheme;

    // Resources loaded from plugins
    this.resources;

    // i18n for themes
    this.i18n = new I18n();
}
util.inherits(_WebsiteOutput, Output);

var WebsiteOutput = conrefsLoader(_WebsiteOutput);

// Name of the generator
// It's being used as a prefix for templates
WebsiteOutput.prototype.name = 'website';

// Load and setup the theme
WebsiteOutput.prototype.prepare = function() {
    var that = this;

    return Promise()
    .then(function() {
        return WebsiteOutput.super_.prototype.prepare.apply(that);
    })

    .then(function() {
        // This list is ordered to give priority to templates in the book
        var searchPaths = _.pluck(that.plugins.list(), 'root');

        // The book itself can contains a "_layouts" folder
        searchPaths.unshift(that.book.root);

        // Load i18n
        _.each(searchPaths.concat().reverse(), function(searchPath) {
            var i18nRoot = path.resolve(searchPath, '_i18n');

            if (!fs.existsSync(i18nRoot)) return;
            that.i18n.load(i18nRoot);
        });

        that.env = new nunjucks.Environment(new FSLoader(_.map(searchPaths, templatesPath)));

        // Add GitBook default filters
        _.each(defaultFilters, function(fn, filter) {
            that.env.addFilter(filter, fn);
        });

        // Translate using _i18n locales
        that.env.addFilter('t', function(s) {
            return that.i18n.t(that.book.config.get('language'), s);
        });

        // Transform an absolute path into a relative path
        // using this.ctx.page.path
        that.env.addFilter('resolveFile', function(href) {
            return location.normalize(that.resolveForPage(this.ctx.file.path, href));
        });

        // Test if a file exists
        that.env.addFilter('fileExists', function(href) {
            return fs.existsSync(that.resolve(href));
        });

        // Transform a '.md' into a '.html' (README -> index)
        that.env.addFilter('contentURL', function(s) {
            return that.toURL(s);
        });

        // Relase path to an asset
        that.env.addFilter('resolveAsset', function(href) {
            href = path.join('gitbook', href);

            // Resolve for current file
            if (this.ctx.file) {
                href = that.resolveForPage(this.ctx.file.path, '/' + href);
            }

            // Use assets from parent
            if (that.book.isLanguageBook()) {
                href = path.join('../', href);
            }

            return location.normalize(href);
        });
    })

    // Copy assets from themes before copying files from book
    .then(function() {
        if (that.book.isLanguageBook()) return;

        // Assets from the book are already copied
        // Copy assets from plugins (start with default plugins)
        return Promise.serie(that.plugins.list().reverse(), function(plugin) {
            // Copy assets only if exists (don't fail otherwise)
            var assetFolder = path.join(plugin.root, '_assets', that.name);
            if (!fs.existsSync(assetFolder)) return;

            that.log.debug.ln('copy assets from theme', assetFolder);
            return fs.copyDir(
                assetFolder,
                that.resolve('gitbook'),
                {
                    deleteFirst: false,
                    overwrite: true,
                    confirm: true
                }
            );
        });
    })

    // Load resources for plugins
    .then(function() {
        return that.plugins.getResources(that.name)
        .then(function(resources) {
            that.resources = resources;
        });
    });
};

// Write a page (parsable file)
WebsiteOutput.prototype.onPage = function(page) {
    var that = this;

    // Parse the page
    return page.toHTML(this)

    // Render the page template with the same context as the json output
    .then(function() {
        return that.render('page', page.getOutputContext(that));
    })

    // Write the HTML file
    .then(function(html) {
        return that.writeFile(
            that.outputPath(page.path),
            html
        );
    });
};

// Finish generation, create ebook using ebook-convert
WebsiteOutput.prototype.finish = function() {
    var that = this;

    return Promise()
    .then(function() {
        return WebsiteOutput.super_.prototype.finish.apply(that);
    })

    // Copy assets from plugins
    .then(function() {
        if (that.book.isLanguageBook()) return;
        return that.plugins.copyResources(that.name, that.resolve('gitbook'));
    })

    // Generate homepage to select languages
    .then(function() {
        if (!that.book.isMultilingual()) return;
        return that.outputMultilingualIndex();
    });
};

// ----- Utilities ----

// Write multi-languages index
WebsiteOutput.prototype.outputMultilingualIndex = function() {
    var that = this;

    return that.render('languages', that.getContext())
    .then(function(html) {
        return that.writeFile(
            'index.html',
            html
        );
    });
};

// Render a template using nunjucks
// Templates are stored in `_layouts` folders
WebsiteOutput.prototype.render = function(tpl, context) {
    var filename = this.templateName(tpl);

    context = _.extend(context, {
        template: {
            self: filename
        },

        plugins: {
            resources: this.resources
        },

        options: this.opts
    });

    return Promise.nfcall(this.env.render.bind(this.env), filename, context);
};

// Return a complete name for a template
WebsiteOutput.prototype.templateName = function(name) {
    return path.join(this.name, name+'.html');
};

module.exports = WebsiteOutput;
