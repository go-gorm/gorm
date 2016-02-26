var util = require('util');
var path = require('path');
var crc = require('crc');

var FolderOutput = require('./folder')();
var Promise = require('../utils/promise');
var fs = require('../utils/fs');
var imagesUtil = require('../utils/images');
var location = require('../utils/location');

var DEFAULT_ASSETS_FOLDER = 'assets';

/*
Mixin to inline all the assets in a book:
    - Outline <svg> tags
    - Download remote images
    - Convert .svg images as png
*/

module.exports = function assetsInliner(Base) {
    Base = Base || FolderOutput;

    function AssetsInliner() {
        Base.apply(this, arguments);

        // Map of svg already converted
        this.svgs = {};
        this.inlineSvgs = {};

        // Map of images already downloaded
        this.downloaded = {};
    }
    util.inherits(AssetsInliner, Base);

    // Output a SVG buffer as a file
    AssetsInliner.prototype.onOutputSVG = function(page, svg) {
        this.log.debug.ln('output svg from', page.path);

        // Convert svg buffer to a png file
        return this.convertSVGBuffer(svg)

            // Return relative path from the page
            .then(function(filename) {
                return page.relative('/' + filename);
            });
    };


    // Output an image as a file
    AssetsInliner.prototype.onOutputImage = function(page, src) {
        var that = this;

        return Promise()

        // Download file if external
        .then(function() {
            if (!location.isExternal(src)) return;

            return that.downloadAsset(src)
            .then(function(_asset) {
                src = '/' + _asset;
            });

        })
        .then(function() {
            // Resolve src to a relative filepath to the book's root
            src = page.resolveLocal(src);

            // Already a PNG/JPG/.. ?
            if (path.extname(src).toLowerCase() != '.svg') {
                return src;
            }

            // Convert SVG to PNG
            return that.convertSVGFile(that.resolve(src));
        })

        // Return relative path from the page
        .then(function(filename) {
            return page.relative(filename);
        });
    };

    // Download an asset if not already download; returns the output file
    AssetsInliner.prototype.downloadAsset = function(src) {
        if (this.downloaded[src]) return Promise(this.downloaded[src]);

        var that = this;
        var ext = path.extname(src);
        var hash = crc.crc32(src).toString(16);

        // Create new file
        return this.createNewFile(DEFAULT_ASSETS_FOLDER, hash + ext)
        .then(function(filename) {
            that.downloaded[src] = filename;

            that.log.debug.ln('downloading asset', src);
            return fs.download(src, that.resolve(filename))
            .thenResolve(filename);
        });
    };

    // Convert a .svg into an .png
    // Return the output filename for the .png
    AssetsInliner.prototype.convertSVGFile = function(src) {
        if (this.svgs[src]) return Promise(this.svgs[src]);

        var that = this;
        var hash = crc.crc32(src).toString(16);

        // Create new file
        return this.createNewFile(DEFAULT_ASSETS_FOLDER, hash + '.png')
        .then(function(filename) {
            that.svgs[src] = filename;

            return imagesUtil.convertSVGToPNG(src, that.resolve(filename))
            .thenResolve(filename);
        });
    };

    // Convert an inline svg into an .png
    // Return the output filename for the .png
    AssetsInliner.prototype.convertSVGBuffer = function(buf) {
        var that = this;
        var hash = crc.crc32(buf).toString(16);

        // Already converted?
        if (this.inlineSvgs[hash]) return Promise(this.inlineSvgs[hash]);

        return this.createNewFile(DEFAULT_ASSETS_FOLDER, hash + '.png')
        .then(function(filename) {
            that.inlineSvgs[hash] = filename;

            return imagesUtil.convertSVGBufferToPNG(buf, that.resolve(filename))
            .thenResolve(filename);
        });
    };

    return AssetsInliner;
};
