var _ = require('lodash');
var url = require('url');
var cheerio = require('cheerio');
var domSerializer = require('dom-serializer');
var slug = require('github-slugid');

var Promise = require('../utils/promise');
var location = require('../utils/location');

// Selector to ignore
var ANNOTATION_IGNORE = '.no-glossary,code,pre,a,script,h1,h2,h3,h4,h5,h6';

function HTMLPipeline(htmlString, opts) {
    _.bindAll(this);

    this.opts = _.defaults(opts || {}, {
        // Called once the description has been found
        onDescription: function(description) { },

        // Calcul new href for a relative link
        onRelativeLink: _.identity,

        // Output an image
        onImage: _.identity,

        // Syntax highlighting
        onCodeBlock: _.identity,

        // Output a svg, if returns null the svg is kept inlined
        onOutputSVG: _.constant(null),

        // Words to annotate
        annotations: [],

        // When an annotation is applied
        onAnnotation: function () { }
    });

    this.$ = cheerio.load(htmlString, {
        // We should parse html without trying to normalize too much
        xmlMode: false,

        // SVG need some attributes to use uppercases
        lowerCaseAttributeNames: false,
        lowerCaseTags: false
    });
}

// Transform a query of elements in the page
HTMLPipeline.prototype._transform = function(query, fn) {
    var that = this;

    var $elements = this.$(query);

    return Promise.serie($elements, function(el) {
        var $el = that.$(el);
        return fn.call(that, $el);
    });
};

// Normalize links
HTMLPipeline.prototype.transformLinks = function() {
    return this._transform('a', function($a) {
        var href = $a.attr('href');
        if (!href) return;

        if (location.isAnchor(href)) {
            // Don't "change" anchor links
        } else if (location.isRelative(href)) {
            // Preserve anchor
            var parsed = url.parse(href);
            var filename = this.opts.onRelativeLink(parsed.pathname);

            $a.attr('href', filename + (parsed.hash || ''));
        } else {
            // External links
            $a.attr('target', '_blank');
        }
    });
};

// Normalize images
HTMLPipeline.prototype.transformImages = function() {
    return this._transform('img', function($img) {
        return Promise(this.opts.onImage($img.attr('src')))
        .then(function(filename) {
            $img.attr('src', filename);
        });
    });
};

// Normalize code blocks
HTMLPipeline.prototype.transformCodeBlocks = function() {
    return this._transform('code', function($code) {
        // Extract language
        var lang = _.chain(
                ($code.attr('class') || '').split(' ')
            )
            .map(function(cl) {
                // Markdown
                if (cl.search('lang-') === 0) return cl.slice('lang-'.length);

                // Asciidoc
                if (cl.search('language-') === 0) return cl.slice('language-'.length);

                return null;
            })
            .compact()
            .first()
            .value();

        var source = $code.text();

        return Promise(this.opts.onCodeBlock(source, lang))
        .then(function(blk) {
            if (blk.html === false) {
                $code.text(blk.body);
            } else {
                $code.html(blk.body);
            }
        });
    });
};

// Add ID to headings
HTMLPipeline.prototype.transformHeadings = function() {
    var that = this;

    this.$('h1,h2,h3,h4,h5,h6').each(function() {
        var $h = that.$(this);

        // Already has an ID?
        if ($h.attr('id')) return;
        $h.attr('id', slug($h.text()));
    });
};

// Outline SVG from the HML
HTMLPipeline.prototype.transformSvgs = function() {
    var that = this;

    return this._transform('svg', function($svg) {
        var content = [
            '<?xml version="1.0" encoding="UTF-8"?>',
            renderDOM(that.$, $svg)
        ].join('\n');

        return Promise(that.opts.onOutputSVG(content))
        .then(function(filename) {
            if (!filename) return;

            $svg.replaceWith(that.$('<img>').attr('src', filename));
        });
    });
};

// Annotate the content
HTMLPipeline.prototype.applyAnnotations = function() {
    var that = this;

    _.each(this.opts.annotations, function(annotation) {
        var searchRegex =  new RegExp( '\\b(' + pregQuote(annotation.name.toLowerCase()) + ')\\b' , 'gi' );

        that.$('*').each(function() {
            var $this = that.$(this);

            if (
                $this.is(ANNOTATION_IGNORE) ||
                $this.parents(ANNOTATION_IGNORE).length > 0
            ) return;

            replaceText(that.$, this, searchRegex, function(match) {
                that.opts.onAnnotation(annotation);

                return '<a href="' + that.opts.onRelativeLink(annotation.href) + '" '
                    + 'class="glossary-term" title="'+_.escape(annotation.description)+'">'
                    + match
                    + '</a>';
            });
        });
    });
};

// Extract page description from html
// This can totally be improved
HTMLPipeline.prototype.extractDescription = function() {
    var $p = this.$('p').first();
    var description = $p.text().trim().slice(0, 155);

    this.opts.onDescription(description);
};

// Write content to the pipeline
HTMLPipeline.prototype.output = function() {
    var that = this;

    return Promise()
    .then(this.extractDescription)
    .then(this.transformImages)
    .then(this.transformHeadings)
    .then(this.transformCodeBlocks)
    .then(this.transformSvgs)
    .then(this.applyAnnotations)

    // Transform of links should be applied after annotations
    // because annotations are created as links
    .then(this.transformLinks)

    .then(function() {
        return renderDOM(that.$);
    });
};


// Render a cheerio DOM as html
function renderDOM($, dom, options) {
    if (!dom && $._root && $._root.children) {
        dom = $._root.children;
    }
    options = options|| dom.options || $._options;
    return domSerializer(dom, options);
}

// Replace text in an element
function replaceText($, el, search, replace, text_only ) {
    return $(el).each(function(){
        var node = this.firstChild,
            val,
            new_val,

            // Elements to be removed at the end.
            remove = [];

        // Only continue if firstChild exists.
        if ( node ) {

            // Loop over all childNodes.
            while (node) {

                // Only process text nodes.
                if ( node.nodeType === 3 ) {

                    // The original node value.
                    val = node.nodeValue;

                    // The new value.
                    new_val = val.replace( search, replace );

                    // Only replace text if the new value is actually different!
                    if ( new_val !== val ) {

                        if ( !text_only && /</.test( new_val ) ) {
                            // The new value contains HTML, set it in a slower but far more
                            // robust way.
                            $(node).before( new_val );

                            // Don't remove the node yet, or the loop will lose its place.
                            remove.push( node );
                        } else {
                            // The new value contains no HTML, so it can be set in this
                            // very fast, simple way.
                            node.nodeValue = new_val;
                        }
                    }
                }

                node = node.nextSibling;
            }
        }

        // Time to remove those elements!
        if (remove.length) $(remove).remove();
    });
}

function pregQuote( str ) {
    return (str+'').replace(/([\\\.\+\*\?\[\^\]\$\(\)\{\}\=\!\<\>\|\:])/g, '\\$1');
}

module.exports = HTMLPipeline;
