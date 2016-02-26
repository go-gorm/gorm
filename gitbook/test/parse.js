var mock = require('./mock');

describe('Parsing', function() {
    it('should not fail without SUMMARY', function() {
        return mock.setupBook({
            'README.md': ''
        })
        .then(function(book) {
            return book.parse().should.be.fulfilled();
        });
    });

    it('should fail without README', function() {
        return mock.setupBook({
            'SUMMARY.md': ''
        })
        .then(function(book) {
            return book.parse().should.be.rejected;
        });
    });

    it('should add GLOSSARY as a page', function() {
        return mock.setupDefaultBook({
            'GLOSSARY.md': ''
        })
        .then(function(book) {
            return book.parse()
            .then(function() {
                book.hasPage('GLOSSARY.md').should.equal(true);
            });
        });
    });

    describe('Multilingual book', function() {
        var book;

        before(function() {
            return mock.setupBook({
                'LANGS.md': '# Languages\n\n'
                    + '* [English](./en)\n'
                    + '* [French](./fr)\n\n',
                'en/README.md': '# English',
                'en/SUMMARY.md': '# Summary',
                'fr/README.md': '# French',
                'fr/SUMMARY.md': '# Summary'
            })
            .then(function(_book) {
                book = _book;
                return book.parse();
            });
        });

        it('should list language books', function() {
            book.isMultilingual().should.equal(true);
            book.books.should.have.lengthOf(2);
        });
    });
});

