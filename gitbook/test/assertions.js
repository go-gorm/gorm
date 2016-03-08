var fs = require('fs');
var path = require('path');
var _ = require('lodash');
var cheerio = require('cheerio');
var should = require('should');

// Assertions to test if an Output has generated a file
should.Assertion.add('file', function(file, description) {
    var rootFolder;
    if (_.isFunction(this.obj.root)) {
        rootFolder = this.obj.root();
    } else {
        rootFolder = this.obj;
    }

    this.params = {
        actual: rootFolder,
        operator: 'have file ' + file,
        message: description
    };

    if (_.isFunction(this.obj.resolve)) {
        file = this.obj.resolve(file);
    } else {
        file = path.resolve(rootFolder, file);
    }

    this.assert(fs.existsSync(file));
});

should.Assertion.add('html', function(rules, description) {
    this.params = { actual: 'HTML string', operator: 'valid html', message: description };
    var $ = cheerio.load(this.obj);

    _.each(rules, function(validations, query) {
        validations = _.defaults(validations || {}, {
            // Select a specific element in the list of matched elements
            index: null,

            // Check that there is the correct count of elements
            count: 1,

            // Check attribute values
            attributes: {},

            // Trim inner text
            trim: false,

            // Check inner text
            text: undefined
        });

        var $el = $(query);

        // Select correct element
        if (_.isNumber(validations.index)) $el = $($el.get(validations.index));

        // Test number of elements
        $el.length.should.be.equal(validations.count);

        // Test text
        if (validations.text !== undefined) {
            var text = $el.text();
            if (validations.trim) text = text.trim();
            text.should.be.equal(validations.text);
        }

        // Test attributes
        _.each(validations.attributes, function(value, name) {
            var attr = $el.attr(name);
            should(attr).be.ok();
            attr.should.be.equal(value);
        });
    });
});
