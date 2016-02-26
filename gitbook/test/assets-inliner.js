var cheerio = require('cheerio');
var path = require('path');

var mock = require('./mock');
var AssetsInliner = require('../lib/output/assets-inliner')();

describe('Assets Inliner Output', function() {
    var output;

    before(function() {
        var SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100" version="1.1"><rect width="200" height="100" stroke="black" stroke-width="6" fill="green"/></svg>';

        return mock.outputDefaultBook(AssetsInliner, {
            'README.md': '',

            // SVGs
            'svg_file.md': '![image](test.svg)',
            'svg_inline.md': 'This is a svg: '+SVG,
            'test.svg': '<?xml version="1.0" encoding="UTF-8"?>' + SVG,

            // Relative
            'folder/test.md': '![image](../test.svg)',

            // Remote images
            'remote_png.md': '![image](https://upload.wikimedia.org/wikipedia/commons/4/47/PNG_transparency_demonstration_1.png)',
            'remote_svg.md': '![image](https://upload.wikimedia.org/wikipedia/commons/0/02/SVG_logo.svg)',

            'SUMMARY.md': '* [svg inline](svg_inline.md)\n' +
                '* [svg file](svg_file.md)\n' +
                '* [remote png file](remote_png.md)\n' +
                '* [remote svg file](remote_svg.md)\n' +
                '* [relative image](folder/test.md)\n' +
                '\n\n'
        })
        .then(function(_output) {
            output = _output;
        });
    });

    function testImageInPage(filename) {
        var page = output.book.getPage(filename);
        var $ = cheerio.load(page.content);

        // Is there an image?
        var $img = $('img');
        $img.length.should.equal(1);

        // Does the file exists
        var src = $img.attr('src');

        // Resolve the filename
        src = page.resolveLocal(src);

        output.should.have.file(src);
        path.extname(src).should.equal('.png');

        return src;
    }

    describe('SVG', function() {
        it('should correctly convert SVG files to PNG', function() {
            testImageInPage('svg_file.md');
        });

        it('should correctly convert inline SVG  to PNG', function() {
            testImageInPage('svg_inline.md');
        });
    });

    describe('Remote Assets', function() {
        it('should correctly download a PNG file', function() {
            testImageInPage('remote_png.md');
        });

        it('should correctly download then convert a remote SVG to PNG', function() {
            testImageInPage('remote_svg.md');
        });
    });

    describe('Relative Images', function() {
        it('should correctly resolve image', function() {
            testImageInPage('folder/test.md');
        });
    });
});

