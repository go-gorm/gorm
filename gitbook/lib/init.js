var path = require('path');

var fs = require('./utils/fs');
var Promise = require('./utils/promise');

// Initialize folder structure for a book
// Read SUMMARY to created the right chapter
function initBook(book) {
    var extensionToUse = '.md';

    book.log.info.ln('init book at', book.root);
    return fs.mkdirp(book.root)
    .then(function() {
        return book.config.load();
    })
    .then(function() {
        book.log.info.ln('detect structure from SUMMARY (if it exists)');
        return book.summary.load();
    })
    .then(function() {
        var summary = book.summary.path || 'SUMMARY.md';
        var articles = book.summary.flatten();

        // Use extension of summary
        extensionToUse = path.extname(summary);

        // Readme doesn't have a path
        if (!articles[0].path) {
            articles[0].path = 'README' + extensionToUse;
        }

        // Summary doesn't exists? create one
        if (!book.summary.path) {
            articles.push({
                title: 'Summary',
                path: 'SUMMARY'+extensionToUse
            });
        }

        // Create files that don't exist
        return Promise.serie(articles, function(article) {
            if (!article.path) return;

            var absolutePath = book.resolve(article.path);

            return fs.exists(absolutePath)
            .then(function(exists) {
                if(exists) {
                    book.log.info.ln('found', article.path);
                    return;
                } else {
                    book.log.info.ln('create', article.path);
                }

                return fs.mkdirp(path.dirname(absolutePath))
                .then(function() {
                    return fs.writeFile(absolutePath, '# '+article.title+'\n\n');
                });
            });
        });
    })
    .then(function() {
        book.log.info.ln('initialization is finished');
    });
}

module.exports = initBook;
