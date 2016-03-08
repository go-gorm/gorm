var _ = require('lodash');
var path = require('path');

var Book = require('../book');
var NodeFS = require('../fs/node');
var Logger = require('../utils/logger');
var Promise = require('../utils/promise');
var fs = require('../utils/fs');
var JSONOutput = require('../output/json');
var WebsiteOutput = require('../output/website');
var EBookOutput = require('../output/ebook');

var nodeFS = new NodeFS();

var LOG_OPTION = {
    name: 'log',
    description: 'Minimum log level to display',
    values: _.chain(Logger.LEVELS)
        .keys()
        .map(function(s) {
            return s.toLowerCase();
        })
        .value(),
    defaults: 'info'
};

var FORMAT_OPTION = {
    name: 'format',
    description: 'Format to build to',
    values: ['website', 'json', 'ebook'],
    defaults: 'website'
};

var FORMATS = {
    json: JSONOutput,
    website: WebsiteOutput,
    ebook: EBookOutput
};

// Commands which is processing a book
// the root of the book is the first argument (or current directory)
function bookCmd(fn) {
    return function(args, kwargs) {
        var input = path.resolve(args[0] || process.cwd());
        var book = new Book({
            fs: nodeFS,
            root: input,
            logLevel: kwargs.log
        });

        return fn(book, args.slice(1), kwargs);
    };
}

// Commands which is working on a Output instance
function outputCmd(fn) {
    return bookCmd(function(book, args, kwargs) {
        var Out = FORMATS[kwargs.format];
        var outputFolder = undefined;

        // Set output folder
        if (args[0]) {
            outputFolder = path.resolve(process.cwd(), args[0]);
        }

        return fn(new Out(book, {
            root: outputFolder
        }), args);
    });
}

// Command to generate an ebook
function ebookCmd(format) {
    return {
        name: format + ' [book] [output] [file]',
        description: 'generates ebook '+format,
        options: [
            LOG_OPTION
        ],
        exec: bookCmd(function(book, args, kwargs) {
            return fs.tmpDir()
            .then(function(dir) {
                var ext = '.'+format;
                var outputFile = path.resolve(process.cwd(), args[0] || ('book' + ext));
                var output = new EBookOutput(book, {
                    root: dir,
                    format: format
                });

                return output.book.parse()
                .then(function() {
                    return output.generate();
                })

                // Copy the ebook files
                .then(function() {
                    if (output.book.isMultilingual()) {
                        return Promise.serie(output.book.langs.list(), function(lang) {
                            var _outputFile = path.join(
                                path.dirname(outputFile),
                                path.basename(outputFile, ext) + '_' + lang.id + ext
                            );

                            return fs.copy(
                                path.resolve(dir, lang.id, 'index' + ext),
                                _outputFile
                            );
                        })
                        .thenResolve(output.book.langs.count());
                    } else {
                        return fs.copy(
                            path.resolve(dir, 'index' + ext),
                            outputFile
                        ).thenResolve(1);
                    }
                })
                .then(function(n) {
                    output.book.log.info.ok(n+' file(s) generated');

                    output.book.log.info('cleaning up... ');
                    return output.book.log.info.promise(fs.rmDir(dir));
                });
            });
        })
    };
}

module.exports = {
    nodeFS: nodeFS,
    bookCmd: bookCmd,
    outputCmd: outputCmd,
    ebookCmd: ebookCmd,

    options: {
        log: LOG_OPTION,
        format: FORMAT_OPTION
    },

    FORMATS: FORMATS
};
