var _ = require('lodash');

function BackboneFile(book) {
    if (!(this instanceof BackboneFile)) return new BackboneFile(book);

    this.book = book;
    this.log = this.book.log;

    // Filename in the book
    this.path = '';
    this.parser;

    _.bindAll(this);
}

// Type of the backbone file
BackboneFile.prototype.type = '';

// Parse a backbone file
BackboneFile.prototype.parse = function() {
    // To be implemented by each child
};

// Handle case where file doesn't exists
BackboneFile.prototype.parseNotFound = function() {

};

// Return true if backbone file exists
BackboneFile.prototype.exists = function() {
    return Boolean(this.path);
};

// Locate a backbone file, could be .md, .asciidoc, etc
BackboneFile.prototype.locate = function() {
    var that = this;
    var filename = this.book.config.getStructure(this.type, true);
    this.log.debug.ln('locating', this.type, ':', filename);

    return this.book.findParsableFile(filename)
    .then(function(result) {
        if (!result) return;

        that.path = result.path;
        that.parser = result.parser;
    });
};

// Read and parse the file
BackboneFile.prototype.load = function() {
    var that = this;
    this.log.debug.ln('loading', this.type, ':', that.path);

    return this.locate()
    .then(function() {
        if (!that.path) return that.parseNotFound();

        that.log.debug.ln(that.type, 'located at', that.path);

        return that.book.readFile(that.path)

        // Parse it
        .then(function(content) {
            return that.parse(content);
        });
    });
};

module.exports = BackboneFile;
