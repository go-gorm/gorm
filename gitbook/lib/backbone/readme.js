var util = require('util');
var BackboneFile = require('./file');

function Readme() {
    BackboneFile.apply(this, arguments);

    this.title;
    this.description;
}
util.inherits(Readme, BackboneFile);

Readme.prototype.type = 'readme';

// Parse the readme content
Readme.prototype.parse = function(content) {
    var that = this;

    return this.parser.readme(content)
    .then(function(out) {
        that.title = out.title;
        that.description = out.description;
    });
};


module.exports = Readme;
