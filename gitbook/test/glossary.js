var should = require('should');
var mock = require('./mock');

describe('Glossary', function() {
    it('should parse empty glossary', function() {
        return mock.setupDefaultBook({
            'GLOSSARY.md': ''
        })
        .then(function(book) {
            return book.config.load()

            .then(function() {
                return book.glossary.load();
            })
            .then(function() {
                book.glossary.isEmpty().should.be.true();
            });
        });
    });

    describe('Non-empty glossary', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook({
                'GLOSSARY.md': '# Glossary\n\n## Hello World\n\nThis is an entry'
            })
            .then(function(_book) {
                book = _book;
                return book.config.load();
            })
            .then(function() {
                return book.glossary.load();
            });
        });

        it('should not be empty', function() {
            book.glossary.isEmpty().should.be.false();
        });

        describe('glossary.get', function() {
            it('should return an existing entry', function() {
                var entry = book.glossary.get('hello_world');
                should.exist(entry);

                entry.name.should.equal('Hello World');
                entry.description.should.equal('This is an entry');
                entry.id.should.equal('hello_world');
            });

            it('should undefined return non existing entry', function() {
                var entry = book.glossary.get('cool');
                should.not.exist(entry);
            });
        });

        describe('glossary.find', function() {
            it('should return an existing entry', function() {
                var entry = book.glossary.find('HeLLo World');
                should.exist(entry);
                entry.id.should.equal('hello_world');
            });

            it('should return undefined for non existing entry', function() {
                var entry = book.glossary.find('Hello');
                should.not.exist(entry);
            });
        });
    });
});

