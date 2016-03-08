var _ = require('lodash');
var Ignore = require('ignore');
var path = require('path');

var Promise = require('../utils/promise');
var pathUtil = require('../utils/path');
var location = require('../utils/location');
var PluginsManager = require('../plugins');
var TemplateEngine = require('../template');

/*
Output is like a stream interface for a parsed book
to output "something".

The process is mostly on the behavior of "onPage" and "onAsset"
*/

function Output(book, opts, parent) {
    _.bindAll(this);
    this.parent = parent;

    this.opts = _.defaults({}, opts || {}, {
        directoryIndex: true
    });

    this.book = book;
    book.output = this;
    this.log = this.book.log;

    // Create plugins manager
    this.plugins = new PluginsManager(this.book);

    // Create template engine
    this.template = new TemplateEngine(this);

    // Files to ignore in output
    this.ignore = Ignore();
}

// Default extension for output
Output.prototype.defaultExtension = '.html';

// Start the generation, for a parsed book
Output.prototype.generate = function() {
    var that = this;
    var isMultilingual = this.book.isMultilingual();

    return Promise()

    // Load all plugins
    .then(function() {
        return that.plugins.loadAll()
        .then(function() {
            that.template.addFilters(that.plugins.getFilters());
            that.template.addBlocks(that.plugins.getBlocks());
        });
    })

    // Transform the configuration
    .then(function() {
        return that.plugins.hook('config', that.book.config.dump())
        .then(function(cfg) {
            that.book.config.replace(cfg);
        });
    })

    // Initialize the generation
    .then(function() {
        return that.plugins.hook('init');
    })
    .then(function() {
        that.log.info.ln('preparing the generation');
        return that.prepare();
    })

    // Process all files
    .then(function() {
        that.log.debug.ln('listing files');
        return that.book.fs.listAllFiles(that.book.root);
    })

    // We want to process assets first, then pages
    // Since pages can have logic based on existance of assets
    .then(function(files) {
        // Split into pages/assets
        var byTypes = _.chain(files)
            .filter(that.ignore.createFilter())

            // Ignore file present in a language book
            .filter(function(filename) {
                return !(isMultilingual && that.book.isInLanguageBook(filename));
            })

            .groupBy(function(filename) {
                return (that.book.hasPage(filename)? 'page' : 'asset');
            })

            .value();

        return Promise.serie(byTypes.asset, function(filename) {
            that.log.debug.ln('copy asset', filename);
            return that.onAsset(filename);
        })
        .then(function() {
            return Promise.serie(byTypes.page, function(filename) {
                that.log.debug.ln('process page', filename);
                return that.onPage(that.book.getPage(filename));
            });
        });
    })

    // Generate sub-books
    .then(function() {
        if (!that.book.isMultilingual()) return;

        return Promise.serie(that.book.books, function(subbook) {
            that.log.info.ln('');
            that.log.info.ln('start generation of language "' + path.relative(that.book.root, subbook.root) + '"');

            var out = that.onLanguageBook(subbook);
            return out.generate();
        });
    })

    // Finish the generation
    .then(function() {
        return that.plugins.hook('finish:before');
    })
    .then(function() {
        that.log.debug.ln('finishing the generation');
        return that.finish();
    })
    .then(function() {
        return that.plugins.hook('finish');
    })

    .then(function() {
        if (!that.book.isLanguageBook()) that.log.info.ln('');
        that.log.info.ok('generation finished with success!');
    });
};

// Prepare the generation
Output.prototype.prepare = function() {
    this.ignore.addPattern(_.compact([
        '.gitignore',
        '.ignore',
        '.bookignore',
        'node_modules',
        '_layouts',

        // The configuration file should not be copied in the output
        this.book.config.path,

        // Structure file to ignore
        this.book.summary.path,
        this.book.langs.path
    ]));
};

// Write a page (parsable file), ex: markdown, etc
Output.prototype.onPage = function(page) {
    return page.toHTML(this);
};

// Copy an asset file (non-parsable), ex: images, etc
Output.prototype.onAsset = function(filename) {

};

// Finish the generation
Output.prototype.finish = function() {

};

// Resolve an HTML link
Output.prototype.onRelativeLink = function(currentPage, href) {
    var to = currentPage.followPage(href);

    // Replace by an .html link
    if (to) {
        href = to.path;

        // Recalcul as relative link
        href = currentPage.relative(href);

        // Replace .md by .html
        href = this.toURL(href);
    }

    return href;
};

// Output a SVG buffer as a file
Output.prototype.onOutputSVG = function(page, svg) {
    return null;
};

// Output an image as a file
// Normalize the relative link
Output.prototype.onOutputImage = function(page, imgFile) {
    imgFile = page.resolveLocal(imgFile);
    return page.relative(imgFile);
};

// Read a template by its source URL
Output.prototype.onGetTemplate = function(sourceUrl) {
    throw new Error('template not found '+sourceUrl);
};

// Generate a source URL for a template
Output.prototype.onResolveTemplate = function(from, to) {
    return path.resolve(path.dirname(from), to);
};

// Prepare output for a language book
Output.prototype.onLanguageBook = function(book) {
    return new this.constructor(book, this.opts, this);
};


// ---- Utilities ----

// Return a default context for templates
Output.prototype.getContext = function() {
    return _.extend(
        {},
        this.book.getContext(),
        this.book.langs.getContext(),
        this.book.summary.getContext(),
        this.book.glossary.getContext(),
        this.book.config.getContext()
    );
};

// Resolve a file path in the context of a specific page
// Result is an "absolute path relative to the output folder"
Output.prototype.resolveForPage = function(page, href) {
    if (_.isString(page)) page = this.book.getPage(page);

    href = page.relative(href);
    return this.onRelativeLink(page, href);
};

// Filename for output
// READMEs are replaced by index.html
// /test/README.md -> /test/index.html
Output.prototype.outputPath = function(filename, ext) {
    ext = ext || this.defaultExtension;
    var output = filename;

    if (
        path.basename(filename, path.extname(filename)) == 'README' ||
        output == this.book.readme.path
    ) {
        output = path.join(path.dirname(output), 'index'+ext);
    } else {
        output = pathUtil.setExtension(output, ext);
    }

    return output;
};

// Filename for output
// /test/index.html -> /test/
Output.prototype.toURL = function(filename, ext) {
    var href = this.outputPath(filename, ext);

    if (path.basename(href) == 'index.html' && this.opts.directoryIndex) {
        href = path.dirname(href) + '/';
    }

    return location.normalize(href);
};

module.exports = Output;
