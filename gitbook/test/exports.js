var should = require('should');
var gitbook = require('../');

describe('Exports', function() {
    it('should export the Book class', function() {
        should(gitbook.Book).be.a.Function();
    });

    it('should export the list of commands', function() {
        should(gitbook.commands).be.an.Array();
    });
});
