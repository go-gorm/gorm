var path = require('path');
var pathUtils = require('../lib/utils/path');

describe('Paths', function() {

    describe('setExtension', function() {
        it('should correctly change extension of filename', function() {
            pathUtils.setExtension('test.md', '.html').should.be.equal('test.html');
            pathUtils.setExtension('test.md', '.json').should.be.equal('test.json');
        });

        it('should correctly change extension of path', function() {
            pathUtils.setExtension('hello/test.md', '.html').should.be.equal(path.normalize('hello/test.html'));
            pathUtils.setExtension('hello/test.md', '.json').should.be.equal(path.normalize('hello/test.json'));
        });
    });
});
