var _ = require('lodash');
var path = require('path');

var markdownParser = require('gitbook-markdown');
var asciidocParser = require('gitbook-asciidoc');

var Promise = require('./utils/promise');

// This list is ordered by priority of parsers to use
var PARSERS = [
    createParser(markdownParser, {
        name: 'markdown',
        extensions: ['.md', '.markdown', '.mdown']
    }),
    createParser(asciidocParser, {
        name: 'asciidoc',
        extensions: ['.adoc', '.asciidoc']
    })
];


// Prepare and compose a parser
function createParser(parser, base) {
    var nparser = base;

    nparser.glossary = Promise.wrapfn(parser.glossary);
    nparser.glossary.toText = Promise.wrapfn(parser.glossary.toText);

    nparser.summary = Promise.wrapfn(parser.summary);
    nparser.summary.toText = Promise.wrapfn(parser.summary.toText);

    nparser.langs = Promise.wrapfn(parser.langs);
    nparser.langs.toText = Promise.wrapfn(parser.langs.toText);

    nparser.readme = Promise.wrapfn(parser.readme);

    nparser.page = Promise.wrapfn(parser.page);
    nparser.page.prepare = Promise.wrapfn(parser.page.prepare || _.identity);

    return nparser;
}

// Return a specific parser according to an extension
function getParser(ext) {
    return _.find(PARSERS, function(input) {
        return input.name == ext || _.contains(input.extensions, ext);
    });
}

// Return parser for a file
function getParserForFile(filename) {
    return getParser(path.extname(filename));
}

module.exports = {
    all: PARSERS,
    extensions: _.flatten(_.pluck(PARSERS, 'extensions')),
    get: getParser,
    getForFile: getParserForFile
};
