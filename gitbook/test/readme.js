var mock = require('./mock');

describe('Readme', function() {
    it('should parse empty readme', function() {
        return mock.setupDefaultBook({
            'README.md': ''
        })
        .then(function(book) {
            return book.config.load()

            .then(function() {
                return book.readme.load();
            });
        });
    });

    it('should parse readme', function() {
        return mock.setupDefaultBook({
            'README.md': '# Hello World\nThis is my book'
        })
        .then(function(book) {
            return book.readme.load()
            .then(function() {
                book.readme.title.should.equal('Hello World');
                book.readme.description.should.equal('This is my book');
            });
        });
    });

    it('should parse AsciiDoc readme', function() {
        return mock.setupBook({
            'README.adoc': '# Hello World\n\nThis is my book\n'
        })
        .then(function(book) {
            return book.readme.load()
            .then(function() {
                book.readme.title.should.equal('Hello World');
                book.readme.description.should.equal('This is my book');
            });
        });
    });
});

