var path = require('path');
var should = require('should');

var mock = require('./mock');

describe('Locate', function() {
    it('should use root folder if no .gitbook', function() {
        return mock.setupBook({
            'README.md': '# Hello'
        })
        .then(function(book) {
            return book.prepareConfig()
            .then(function() {
                should(book.originalRoot).not.be.ok();
            });
        });
    });

    it('should use resolve using book.js root property', function() {
        return mock.setupBook({
            'README.md': '# Hello',
            'docs/README.md': '# Hello Book',
            'book.json': { root: './docs' }
        })
        .then(function(book) {
            return book.prepareConfig()
            .then(function() {
                should(book.originalRoot).be.ok();
                book.root.should.equal(path.resolve(book.originalRoot, 'docs'));
            });
        });
    });

});
