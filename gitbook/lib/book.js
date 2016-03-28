var _ = require('lodash');
var path = require('path');
var Ignore = require('ignore');

var Config = require('./config');
var Readme = require('./backbone/readme');
var Glossary = require('./backbone/glossary');
var Summary = require('./backbone/summary');
var Langs = require('./backbone/langs');
var Page = require('./page');
var pathUtil = require('./utils/path');
var error = require('./utils/error');
var Promise = require('./utils/promise');
var Logger = require('./utils/logger');
var parsers = require('./parsers');
var initBook = require('./init');


/*
The Book class is an interface for parsing books content.
It does not require to run on Node.js, isnce it only depends on the fs implementation
*/

function Book(opts) {
    if (!(this instanceof Book)) return new Book(opts);

    this.opts = _.defaults(opts || {}, {
        fs: null,

        // Root path for the book
        root: '',

        // Extend book configuration
        config: {},

        // Log function
        log: function(msg) {
            process.stdout.write(msg);
        },

        // Log level
        logLevel: 'info'
    });

    if (!opts.fs) throw error.ParsingError(new Error('Book requires a fs instance'));

    // Root path for the book
    this.root = opts.root;

    // If multi-lingual, book can have a parent
    this.parent = opts.parent;
    if (this.parent) {
        this.language = path.relative(this.parent.root, this.root);
    }

    // A book is linked to an fs, to access its content
    this.fs = opts.fs;

    // Rules to ignore some files
    this.ignore = Ignore();
    this.ignore.addPattern([
        // Skip Git stuff
        '.git/',

        // Skip OS X meta data
        '.DS_Store',

        // Skip stuff installed by plugins
        'node_modules',

        // Skip book outputs
        '_book',
        '*.pdf',
        '*.epub',
        '*.mobi'
    ]);

    // Create a logger for the book
    this.log = new Logger(opts.log, opts.logLevel);

    // Create an interface to access the configuration
    this.config = new Config(this, opts.config);

    // Interfaces for the book structure
    this.readme = new Readme(this);
    this.summary = new Summary(this);
    this.glossary = new Glossary(this);

    // Multilinguals book
    this.langs = new Langs(this);
    this.books = [];

    // List of page in the book
    this.pages = {};

    // Deprecation for templates
    Object.defineProperty(this, 'options', {
        get: function () {
            this.log.warn.ln('"options" property is deprecated, use config.get(key) instead');
            var cfg = this.config.dump();
            error.deprecateField(cfg, 'book', (this.output? this.output.name : null), '"options.generator" property is deprecated, use "output.name" instead');

            // options.generator
            cfg.generator = this.output? this.output.name : null;

            // options.output
            cfg.output = this.output? this.output.root() : null;

            return cfg;
        }
    });

    _.bindAll(this);

    // Loop for template filters/blocks
    error.deprecateField(this, 'book', this, '"book" property is deprecated, use "this" directly instead');
}

// Return templating context for the book
Book.prototype.getContext = function() {
    var variables = this.config.get('variables', {});

    return {
        book: _.extend({
            language: this.language
        }, variables)
    };
};

// Parse and prepare the configuration, fail if invalid
Book.prototype.prepareConfig = function() {
    var that = this;

    return this.config.load()
    .then(function() {
        var rootFolder = that.config.get('root');
        if (!rootFolder) return;

        that.originalRoot = that.root;
        that.root = path.resolve(that.root, rootFolder);
    });
};

// Resolve a path in the book source
// Enforce that the output path is in the scope
Book.prototype.resolve = function() {
    var filename = path.resolve.apply(path, [this.root].concat(_.toArray(arguments)));
    if (!this.isFileInScope(filename)) {
        throw error.FileOutOfScopeError({
            filename: filename,
            root: this.root
        });
    }

    return filename;
};

// Return false if a file is outside the book' scope
Book.prototype.isFileInScope = function(filename) {
    filename = path.resolve(this.root, filename);

    // Is the file in the scope of the parent?
    if (this.parent && this.parent.isFileInScope(filename)) return true;

    // Is file in the root folder?
    return pathUtil.isInRoot(this.root, filename);
};

// Parse .gitignore, etc to extract rules
Book.prototype.parseIgnoreRules = function() {
    var that = this;

    return Promise.serie([
        '.ignore',
        '.gitignore',
        '.bookignore'
    ], function(filename) {
        return that.readFile(filename)
        .then(function(content) {
            that.ignore.addPattern(content.toString().split(/\r?\n/));
        }, function() {
            return Promise();
        });
    });
};

// Parse the whole book
Book.prototype.parse = function() {
    var that = this;

    return Promise()
    .then(this.prepareConfig)
    .then(this.parseIgnoreRules)

    // Parse languages
    .then(function() {
        return that.langs.load();
    })

    .then(function() {
        if (that.isMultilingual()) {
            if (that.isLanguageBook()) {
                throw error.ParsingError(new Error('A multilingual book as a language book is forbidden'));
            }

            that.log.info.ln('Parsing multilingual book, with', that.langs.count(), 'languages');

            // Create a new book for each language and parse it
            return Promise.serie(that.langs.list(), function(lang) {
                that.log.debug.ln('Preparing book for language', lang.id);
                var langBook = new Book(_.extend({}, that.opts, {
                    parent: that,
                    config: that.config.dump(),
                    root: that.resolve(lang.id)
                }));

                that.books.push(langBook);

                return langBook.parse();
            });
        }

        return Promise()

        // Parse the readme
        .then(that.readme.load)
        .then(function() {
            if (!that.readme.exists()) {
                throw new error.FileNotFoundError({ filename: 'README' });
            }

            // Default configuration to infos extracted from readme
            if (!that.config.get('title')) that.config.set('title', that.readme.title);
            if (!that.config.get('description')) that.config.set('description', that.readme.description);
        })

        // Parse the summary
        .then(that.summary.load)
        .then(function() {
            if (!that.summary.exists()) {
                that.log.warn.ln('no summary file in this book');
            }

            // Index summary's articles
            that.summary.walk(function(article) {
                if (!article.hasLocation() || article.isExternal()) return;
                that.addPage(article.path);
            });
        })

        // Parse the glossary
        .then(that.glossary.load)

        // Add the glossary as a page
        .then(function() {
            if (!that.glossary.exists()) return;
            that.addPage(that.glossary.path);
        });
    });
};

// Mark a filename as being parsable
Book.prototype.addPage = function(filename) {
    if (this.hasPage(filename)) return this.getPage(filename);

    filename = pathUtil.normalize(filename);
    this.pages[filename] = new Page(this, filename);
    return this.pages[filename];
};

// Return a page by its filename (or undefined)
Book.prototype.getPage = function(filename) {
    filename = pathUtil.normalize(filename);
    return this.pages[filename];
};


// Return true, if has a specific page
Book.prototype.hasPage = function(filename) {
    return Boolean(this.getPage(filename));
};

// Test if a file is ignored, return true if it is
Book.prototype.isFileIgnored = function(filename) {
    return this.ignore.filter([filename]).length == 0;
};

// Read a file in the book, throw error if ignored
Book.prototype.readFile = function(filename) {
    if (this.isFileIgnored(filename)) return Promise.reject(new error.FileNotFoundError({ filename: filename }));
    return this.fs.readAsString(this.resolve(filename));
};

// Get stat infos about a file
Book.prototype.statFile = function(filename) {
    if (this.isFileIgnored(filename)) return Promise.reject(new error.FileNotFoundError({ filename: filename }));
    return this.fs.stat(this.resolve(filename));
};

// Find a parsable file using a filename
Book.prototype.findParsableFile = function(filename) {
    var that = this;

    var ext = path.extname(filename);
    var basename = path.basename(filename, ext);

    // Ordered list of extensions to test
    var exts = parsers.extensions;
    if (ext) exts = _.uniq([ext].concat(exts));

    return _.reduce(exts, function(prev, ext) {
        return prev.then(function(output) {
            // Stop if already find a parser
            if (output) return output;

            var filepath = basename+ext;

            return that.fs.findFile(that.root, filepath)
            .then(function(realFilepath) {
                if (!realFilepath) return null;

                return {
                    parser: parsers.getByExt(ext),
                    path: realFilepath
                };
            });
        });
    }, Promise(null));
};

// Return true if book is associated to a language
Book.prototype.isLanguageBook = function() {
    return Boolean(this.parent);
};
Book.prototype.isSubBook = Book.prototype.isLanguageBook;

// Return true if the book is main instance of a multilingual book
Book.prototype.isMultilingual = function() {
    return this.langs.count() > 0;
};

// Return true if file is in the scope of this book
Book.prototype.isInBook = function(filename) {
    return pathUtil.isInRoot(
        this.root,
        filename
    );
};

// Return true if file is in the scope of a child book
Book.prototype.isInLanguageBook = function(filename) {
    var that = this;

    return _.some(this.langs.list(), function(lang) {
        return pathUtil.isInRoot(
            that.resolve(lang.id),
            that.resolve(filename)
        );
    });
};

// ----- Parser Methods

// Render a markup string in inline mode
Book.prototype.renderInline = function(type, src) {
    var parser = parsers.get(type);
    return parser.inline(src)
        .get('content');
};

// Render a markup string in block mode
Book.prototype.renderBlock = function(type, src) {
    var parser = parsers.get(type);
    return parser.page(src)
        .get('content');
};


// ----- DEPRECATED METHODS

Book.prototype.contentLink = error.deprecateMethod(function(s) {
    return this.output.toURL(s);
}, '.contentLink() is deprecated, use ".output.toURL()" instead');

Book.prototype.contentPath = error.deprecateMethod(function(s) {
    return this.output.toURL(s);
}, '.contentPath() is deprecated, use ".output.toURL()" instead');

Book.prototype.isSubBook = error.deprecateMethod(function() {
    return this.isLanguageBook();
}, '.isSubBook() is deprecated, use ".isLanguageBook()" instead');


// Initialize a book
Book.init = function(fs, root, opts) {
    var book = new Book(_.extend(opts || {}, {
        root: root,
        fs: fs
    }));

    return initBook(book);
};


module.exports = Book;
