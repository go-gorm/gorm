var _ = require('lodash');
var util = require('util');
var juice = require('juice');

var command = require('../utils/command');
var fs = require('../utils/fs');
var Promise = require('../utils/promise');
var error = require('../utils/error');
var WebsiteOutput = require('./website');
var assetsInliner = require('./assets-inliner');

function _EbookOutput() {
    WebsiteOutput.apply(this, arguments);

    // ebook-convert does not support link like "./"
    this.opts.directoryIndex = false;
}
util.inherits(_EbookOutput, WebsiteOutput);

var EbookOutput = assetsInliner(_EbookOutput);

EbookOutput.prototype.name = 'ebook';

// Return context for templating
// Incldue type of ebbook generated
EbookOutput.prototype.getSelfContext = function() {
    var ctx = EbookOutput.super_.prototype.getSelfContext.apply(this);
    ctx.format = this.opts.format;

    return ctx;
};

// Finish generation, create ebook using ebook-convert
EbookOutput.prototype.finish = function() {
    var that = this;
    if (that.book.isMultilingual()) {
        return EbookOutput.super_.prototype.finish.apply(that);
    }

    return Promise()
    .then(function() {
        return EbookOutput.super_.prototype.finish.apply(that);
    })

    // Generate SUMMARY.html
    .then(function() {
        return that.render('summary', that.getContext())
        .then(function(html) {
            return that.writeFile(
                'SUMMARY.html',
                html
            );
        });
    })

    // Start ebook-convert
    .then(function() {
        return that.ebookConvertOption();
    })

    .then(function(options) {
        if (!that.opts.format) return;

        var cmd = [
            'ebook-convert',
            that.resolve('SUMMARY.html'),
            that.resolve('index.'+that.opts.format),
            command.optionsToShellArgs(options)
        ].join(' ');

        return command.exec(cmd)
        .progress(function(data) {
            that.book.log.debug(data);
        })
        .fail(function(err) {
            if (err.code == 127) {
                throw error.RequireInstallError({
                    cmd: 'ebook-convert',
                    install: 'Install it from Calibre: https://calibre-ebook.com'
                });
            }

            throw error.EbookError(err);
        });
    });
};

// Generate header/footer for PDF
EbookOutput.prototype.getPDFTemplate = function(tpl) {
    var that = this;
    var context = _.extend(
        {
            // Nunjucks context mapping to ebook-convert templating
            page: {
                num: '_PAGENUM_',
                title: '_TITLE_',
                section: '_SECTION_'
            }
        },
        this.getContext()
    );

    return this.render('pdf_'+tpl, context)

    // Inline css, include css relative to the output folder
    .then(function(output) {
        return Promise.nfcall(juice.juiceResources, output, {
            webResources: {
                relativeTo: that.root()
            }
        });
    });
};

// Locate the cover file to use
// Use configuration or search a "cover.jpg" file
// For multi-lingual book, it can use the one from the main book
EbookOutput.prototype.locateCover = function() {
    var cover = this.book.config.get('cover', 'cover.jpg');

    // Resolve to absolute
    cover = this.resolve(cover);

    // Cover doesn't exist and multilingual?
    if (!fs.existsSync(cover)) {
        if (this.parent) return this.parent.locateCover();
        else return undefined;
    }

    return cover;
};

// Generate options for ebook-convert
EbookOutput.prototype.ebookConvertOption = function() {
    var that = this;

    var options = {
        '--cover': this.locateCover(),
        '--title': that.book.config.get('title'),
        '--comments': that.book.config.get('description'),
        '--isbn': that.book.config.get('isbn'),
        '--authors': that.book.config.get('author'),
        '--language': that.book.config.get('language'),
        '--book-producer': 'GitBook',
        '--publisher': 'GitBook',
        '--chapter': 'descendant-or-self::*[contains(concat(\' \', normalize-space(@class), \' \'), \' book-chapter \')]',
        '--level1-toc': 'descendant-or-self::*[contains(concat(\' \', normalize-space(@class), \' \'), \' book-chapter-1 \')]',
        '--level2-toc': 'descendant-or-self::*[contains(concat(\' \', normalize-space(@class), \' \'), \' book-chapter-2 \')]',
        '--level3-toc': 'descendant-or-self::*[contains(concat(\' \', normalize-space(@class), \' \'), \' book-chapter-3 \')]',
        '--no-chapters-in-toc': true,
        '--max-levels': '1',
        '--breadth-first': true
    };

    if (that.opts.format == 'epub') {
        options = _.extend(options, {
            '--dont-split-on-page-breaks': true
        });
    }

    if (that.opts.format != 'pdf') return Promise(options);

    var pdfOptions = that.book.config.get('pdf');

    options = _.extend(options, {
        '--chapter-mark': String(pdfOptions.chapterMark),
        '--page-breaks-before': String(pdfOptions.pageBreaksBefore),
        '--margin-left': String(pdfOptions.margin.left),
        '--margin-right': String(pdfOptions.margin.right),
        '--margin-top': String(pdfOptions.margin.top),
        '--margin-bottom': String(pdfOptions.margin.bottom),
        '--pdf-default-font-size': String(pdfOptions.fontSize),
        '--pdf-mono-font-size': String(pdfOptions.fontSize),
        '--paper-size': String(pdfOptions.paperSize),
        '--pdf-page-numbers': Boolean(pdfOptions.pageNumbers),
        '--pdf-header-template': that.getPDFTemplate('header'),
        '--pdf-footer-template': that.getPDFTemplate('footer'),
        '--pdf-sans-family': String(pdfOptions.fontFamily)
    });

    return that.getPDFTemplate('header')
    .then(function(tpl) {
        options['--pdf-header-template'] = tpl;

        return that.getPDFTemplate('footer');
    })
    .then(function(tpl) {
        options['--pdf-footer-template'] = tpl;

        return options;
    });
};

// Don't write multi-lingual index for wbook
EbookOutput.prototype.outputMultilingualIndex = function() {

};

module.exports = EbookOutput;
