var path = require('path');

var Book = require('../').Book;
var mock = require('./mock');

describe('Locate', function() {
    it('should use root folder if no .gitbook', function() {
        return mock.setupFS({
            'README.md': '# Hello'
        })
        .then(function(root) {
            return Book.locate(mock.fs, root)
                .should.be.fulfilledWith(root);
        });
    });

    it('should use resolve using .gitbook', function() {
        return mock.setupFS({
            'README.md': '# Hello',
            '.gitbook': './docs'
        })
        .then(function(root) {
            return Book.locate(mock.fs, root)
                .should.be.fulfilledWith(path.resolve(root, 'docs'));
        });
    });

});
