var _ = require('lodash');
var path = require('path');
var direction = require('direction');
var fm = require('front-matter');

var error = require('../utils/error');
var pathUtil = require('../utils/path');
var location = require('../utils/location');
var parsers = require('../parsers');
var gitbook = require('../gitbook');
var pluginCompatibility = require('../plugins/compatibility');
var HTMLPipeline = require('./html');

/*
A page represent a parsable file in the book (Markdown, Asciidoc, etc)
*/

function Page(book, filename) {
    if (!(this instanceof Page)) return new Page(book, filename);
    var extension;
    _.bindAll(this);

    this.book = book;
    this.log = this.book.log;

    // Current content
    this.content = '';

    // Short description for the page
    this.description = '';

    // Relative path to the page
    this.path = location.normalize(filename);

    // Absolute path to the page
    this.rawPath = this.book.resolve(filename);

    // Last modification date
    this.mtime = 0;

    // Can we parse it?
    extension = path.extname(this.path);
    this.parser = parsers.get(extension);
    if (!this.parser) throw error.ParsingError(new Error('Can\'t parse file "'+this.path+'"'));

    this.type = this.parser.name;
}

// Return the filename of the page with another extension
// "README.md" -> "README.html"
Page.prototype.withExtension = function(ext) {
    return pathUtil.setExtension(this.path, ext);
};

// Resolve a filename relative to this page
// It returns a path relative to the book root folder
Page.prototype.resolveLocal = function() {
    var dir = path.dirname(this.path);
    var file = path.join.apply(path, _.toArray(arguments));

    return location.toAbsolute(file, dir, '');
};

// Resolve a filename relative to this page
// It returns an absolute path for the FS
Page.prototype.resolve = function() {
    return this.book.resolve(this.resolveLocal.apply(this, arguments));
};

// Convert an absolute path (in the book) to a relative path from this page
Page.prototype.relative = function(name) {
    // Convert /test.png -> test.png
    name = location.toAbsolute(name, '', '');

    return location.relative(
        this.resolve('.') + '/',
        this.book.resolve(name)
    );
};

// Return a page result of a relative page from this page
Page.prototype.followPage = function(filename) {
    var absPath = this.resolveLocal(filename);
    return this.book.getPage(absPath);
};

// Update content of the page
Page.prototype.update = function(content) {
    this.content = content;
};

// Read the page as a string
Page.prototype.read = function() {
    var that = this;

    return this.book.statFile(this.path)
    .then(function(stat) {
        that.mtime = stat.mtime;
        return that.book.readFile(that.path);
    })
    .then(this.update);
};

// Return templating context for this page
// This is used both for themes and page parsing
Page.prototype.getContext = function() {
    var article = this.book.summary.getArticle(this);
    var next = article? article.next() : null;
    var prev = article? article.prev() : null;

    // Detect text direction in this page
    var dir = this.book.config.get('direction');
    if (!dir) {
        dir = direction(this.content);
        if (dir == 'neutral') dir = null;
    }

    return _.extend(
        {
            file: {
                path: this.path,
                mtime: this.mtime,
                type: this.type
            },
            page: {
                title: article? article.title : null,
                description: this.description,
                next: next? next.getContext() : null,
                previous: prev? prev.getContext() : null,
                level: article? article.level : null,
                depth: article? article.depth : 0,
                content: this.content,
                dir: dir
            }
        },
        gitbook.getContext(),
        this.book.getContext(),
        this.book.langs.getContext(),
        this.book.summary.getContext(),
        this.book.glossary.getContext(),
        this.book.config.getContext()
    );
};

// Parse the page and return its content
Page.prototype.toHTML = function(output) {
    var that = this;

    this.log.debug.ln('start parsing file', this.path);

    // Call a hook in the output
    // using an utility to "keep" compatibility with gitbook 2
    function hook(name) {
        return pluginCompatibility.pageHook(that, function(ctx) {
            return output.plugins.hook(name, ctx);
        })
        .then(function(result) {
            if(_.isString(result)) that.update(result);
        });
    }

    return this.read()

    // Parse yaml front matter
    .then(function() {
        var parsed = fm(that.content);

        // Extend page with the fontmatter attribute
        that.description = parsed.attributes.description || '';

        // Keep only the body
        that.update(parsed.body);
    })

    .then(function() {
        return hook('page:before');
    })

    // Pre-process page with parser
    .then(function() {
        return that.parser.page.prepare(that.content)
        .then(that.update);
    })

    // Render template
    .then(function() {
        return output.template.render(that.content, that.getContext(), {
            path: that.path
        })
        .then(that.update);
    })

    // Render markup using the parser
    .then(function() {
        return that.parser.page(that.content)
        .then(function(out) {
            that.update(out.content);
        });
    })

    // Post process templating
    .then(function() {
        return output.template.postProcess(that.content)
        .then(that.update);
    })

    // Normalize HTML output
    .then(function() {
        var pipelineOpts = {
            onRelativeLink: _.partial(output.onRelativeLink, that),
            onImage: _.partial(output.onOutputImage, that),
            onOutputSVG: _.partial(output.onOutputSVG, that),

            // Use 'code' template block
            onCodeBlock: function(source, lang) {
                return output.template.applyBlock('code', {
                    body: source,
                    kwargs: {
                        language: lang
                    }
                });
            },

            // Extract description from page's content if no frontmatter
            onDescription: function(description) {
                if (that.description) return;
                that.description = description;
            },

            // Convert glossary entries to annotations
            annotations: that.book.glossary.annotations()
        };
        var pipeline = new HTMLPipeline(that.content, pipelineOpts);

        return pipeline.output()
        .then(that.update);
    })

    .then(function() {
        return hook('page');
    })

    // Return content itself
    .then(function() {
        return that.content;
    });
};


module.exports = Page;
