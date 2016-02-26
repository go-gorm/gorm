var conrefsLoader = require('./conrefs');

var JSONOutput = conrefsLoader();

JSONOutput.prototype.name = 'json';

// Don't copy asset on JSON output
JSONOutput.prototype.onAsset = function(filename) {};

// Write a page (parsable file)
JSONOutput.prototype.onPage = function(page) {
    var that = this;

    // Parse the page
    return page.toHTML(this)

    // Write as json
    .then(function() {
        var json = page.getContext();

        // Delete some private properties
        delete json.config;

        // Specify JSON output version
        json.version = '2';

        return that.writeFile(
            page.withExtension('.json'),
            JSON.stringify(json, null, 4)
        );
    });
};

// At the end of generation, generate README.json for multilingual books
JSONOutput.prototype.finish = function() {
    if (!this.book.isMultilingual()) return;

    // Copy README.json from main book
    var mainLanguage = this.book.langs.getDefault().id;
    return this.copyFile(
        this.resolve(mainLanguage, 'README.json'),
        'README.json'
    );
};


module.exports = JSONOutput;
