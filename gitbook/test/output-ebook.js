var mock = require('./mock');
var EbookOutput = require('../lib/output/ebook');

describe('Ebook Output', function() {
    describe('Sample Book', function() {
        var output;

        before(function() {
            return mock.outputDefaultBook(EbookOutput)
            .then(function(_output) {
                output = _output;
            });
        });

        it('should correctly generate an index.html', function() {
            output.should.have.file('index.html');
        });

        it('should correctly generate a SUMMARY.html', function() {
            output.should.have.file('index.html');
        });

        it('should correctly copy assets', function() {
            output.should.have.file('gitbook/ebook.css');
        });

        it('should correctly copy plugins', function() {
            output.should.have.file('gitbook/gitbook-plugin-highlight/ebook.css');
        });

    });

});
