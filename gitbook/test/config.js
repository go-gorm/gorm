var should = require('should');
var mock = require('./mock');
var validator = require('../lib/config/validator');

describe('Configuration', function() {

    describe('Validation', function() {
        it('should merge default', function() {
            validator.validate({}).should.have.property('gitbook').which.equal('*');
        });

        it('should throw error for invalid configuration', function() {
            should.throws(function() {
                validator.validate({
                    direction: 'invalid'
                });
            });
        });

        it('should not throw error for non existing configuration', function() {
            validator.validate({
                style: {
                    'pdf': 'test.css'
                }
            });
        });

        it('should validate plugins as an array', function() {
            validator.validate({
                plugins: ['hello']
            });
        });

        it('should validate plugins as a string', function() {
            validator.validate({
                plugins: 'hello,world'
            });
        });

    });

    describe('No configuration', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook()
            .then(function(_book) {
                book = _book;
                return book.config.load();
            });
        });

        it('should signal that configuration is not defined', function() {
            book.config.exists().should.not.be.ok();
        });
    });

    describe('JSON file', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook({
                'book.json': { title: 'Hello World' }
            })
            .then(function(_book) {
                book = _book;
                return book.config.load();
            });
        });

        it('should correctly extend configuration', function() {
            book.config.get('title', '').should.equal('Hello World');
        });
    });

    describe('JS file', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook({
                'book.js': 'module.exports = { title: "Hello World" };'
            })
            .then(function(_book) {
                book = _book;
                return book.config.load();
            });
        });

        it('should correctly extend configuration', function() {
            book.config.get('title', '').should.equal('Hello World');
        });
    });

    describe('Multilingual', function() {
        var book;

        before(function() {
            return mock.setupDefaultBook({
                'book.json': {
                    title: 'Hello World',
                    pluginsConfig: {
                        'test': {
                            'hello': true
                        }
                    }
                },
                'LANGS.md': '# Languages\n\n'
                    + '* [en](./en)\n'
                    + '* [fr](./fr)\n\n',
                'en/README.md': '# Hello',
                'fr/README.md': '# Bonjour',
                'en/book.json': { description: 'In english' },
                'fr/book.json': { description: 'En francais' }
            })
            .then(function(_book) {
                book = _book;
                return book.parse();
            });
        });

        it('should correctly extend configuration', function() {
            book.config.get('title', '').should.equal('Hello World');
            book.config.get('description', '').should.equal('');

            var en = book.books[0];
            en.config.get('title', '').should.equal('Hello World');
            en.config.get('description', '').should.equal('In english');
            en.config.get('pluginsConfig.test.hello').should.equal(true);

            var fr = book.books[1];
            fr.config.get('title', '').should.equal('Hello World');
            fr.config.get('description', '').should.equal('En francais');
            fr.config.get('pluginsConfig.test.hello').should.equal(true);
        });
    });
});

