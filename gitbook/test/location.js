var location = require('../lib/utils/location');

describe('Location', function() {
    it('should correctly test external location', function() {
        location.isExternal('http://google.fr').should.be.exactly(true);
        location.isExternal('https://google.fr').should.be.exactly(true);
        location.isExternal('test.md').should.be.exactly(false);
        location.isExternal('folder/test.md').should.be.exactly(false);
        location.isExternal('/folder/test.md').should.be.exactly(false);
    });

    it('should correctly detect anchor location', function() {
        location.isAnchor('#test').should.be.exactly(true);
        location.isAnchor(' #test').should.be.exactly(true);
        location.isAnchor('https://google.fr#test').should.be.exactly(false);
        location.isAnchor('test.md#test').should.be.exactly(false);
    });

    describe('.relative', function() {
        it('should resolve to a relative path (same folder)', function() {
            location.relative('links/', 'links/test.md').should.equal('test.md');
        });

        it('should resolve to a relative path (parent folder)', function() {
            location.relative('links/', 'test.md').should.equal('../test.md');
        });

        it('should resolve to a relative path (child folder)', function() {
            location.relative('links/', 'links/hello/test.md').should.equal('hello/test.md');
        });
    });

    describe('.toAbsolute', function() {
        it('should correctly transform as absolute', function() {
            location.toAbsolute('http://google.fr').should.be.equal('http://google.fr');
            location.toAbsolute('test.md', './', './').should.be.equal('test.md');
            location.toAbsolute('folder/test.md', './', './').should.be.equal('folder/test.md');
        });

        it('should correctly handle windows path', function() {
            location.toAbsolute('folder\\test.md', './', './').should.be.equal('folder/test.md');
        });

        it('should correctly handle absolute path', function() {
            location.toAbsolute('/test.md', './', './').should.be.equal('test.md');
            location.toAbsolute('/test.md', 'test', 'test').should.be.equal('../test.md');
            location.toAbsolute('/sub/test.md', 'test', 'test').should.be.equal('../sub/test.md');
            location.toAbsolute('/test.png', 'folder', '').should.be.equal('test.png');
        });

        it('should correctly handle absolute path (windows)', function() {
            location.toAbsolute('\\test.png', 'folder', '').should.be.equal('test.png');
        });
    });
});
