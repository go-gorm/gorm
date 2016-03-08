var mock = require('./mock');

describe('Langs', function() {
    it('should parse empty langs', function() {
        return mock.setupDefaultBook({
            'LANGS.md': ''
        })
        .then(function(book) {
            return book.prepareConfig()

            .then(function() {
                return book.langs.load();
            })

            .then(function() {
                book.langs.count().should.equal(0);
            });
        });
    });

    describe('Non-empty languages list', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook({
                'LANGS.md': '# Languages\n\n'
                    + '* [en](./en)\n'
                    + '* [fr](./fr)\n\n'
            })
            .then(function(_book) {
                book = _book;

                return book.langs.load();
            });
        });

        it('should correctly count languages', function() {
            book.langs.count().should.equal(2);
        });

        it('should correctly define book as multilingual', function() {
            book.isMultilingual().should.equal(true);
        });
    });
});

