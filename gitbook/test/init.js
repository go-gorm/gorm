var Book = require('../lib/book');
var mock = require('./mock');

describe('Init', function() {

    it('should create file according to summary', function() {
        return mock.setupFS({
            'SUMMARY.md': '# Summary\n\n'
                + '* [Hello](hello.md)\n'
                + '* [Hello 2](hello 2.md)\n'
        })
        .then(function(rootFolder) {
            return Book.init(mock.fs, rootFolder, {
                log: function() {}
            })
            .then(function() {
                rootFolder.should.have.file('SUMMARY.md');
                rootFolder.should.have.file('README.md');
                rootFolder.should.have.file('hello.md');
                rootFolder.should.have.file('hello 2.md');
            });
        });
    });

    it('should create file subfolder', function() {
        return mock.setupFS({
            'SUMMARY.md': '# Summary\n\n'
                + '* [Hello](test/hello.md)\n'
                + '* [Hello 2](test/test2/world.md)\n'
        })
        .then(function(rootFolder) {
            return Book.init(mock.fs, rootFolder, {
                log: function() {}
            })
            .then(function() {
                rootFolder.should.have.file('README.md');
                rootFolder.should.have.file('SUMMARY.md');
                rootFolder.should.have.file('test/hello.md');
                rootFolder.should.have.file('test/test2/world.md');
            });
        });
    });

    it('should create SUMMARY if non-existant', function() {
        return mock.setupFS({})
        .then(function(rootFolder) {
            return Book.init(mock.fs, rootFolder, {
                log: function() {}
            })
            .then(function() {
                rootFolder.should.have.file('SUMMARY.md');
                rootFolder.should.have.file('README.md');
            });
        });
    });

});
