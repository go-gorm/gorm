var _ = require('lodash');
var path = require('path');
var util = require('util');
var BackboneFile = require('./file');

function Language(title, folder) {
    var that = this;

    this.title = title;
    this.folder = folder;

    Object.defineProperty(this, 'id', {
        get: function() {
            return path.basename(that.folder);
        }
    });
}

/*
A Langs is a list of languages stored in a LANGS.md file
*/
function Langs() {
    BackboneFile.apply(this, arguments);

    this.languages = [];
}
util.inherits(Langs, BackboneFile);

Langs.prototype.type = 'langs';

// Parse the readme content
Langs.prototype.parse = function(content) {
    var that = this;

    return this.parser.langs(content)
    .then(function(langs) {
        that.languages = _.map(langs, function(entry) {
            return new Language(entry.title, entry.path);
        });
    });
};

// Return the list of languages
Langs.prototype.list = function() {
    return this.languages;
};

// Return default/main language for the book
Langs.prototype.getDefault = function() {
    return _.first(this.languages);
};

// Return true if a language is the default one
// "lang" cam be a string (id) or a Language entry
Langs.prototype.isDefault = function(lang) {
    lang = lang.id || lang;
    return (this.cound() > 0 && this.getDefault().id == lang);
};

// Return the count of languages
Langs.prototype.count = function() {
    return _.size(this.languages);
};

// Return templating context for the languages list
Langs.prototype.getContext = function() {
    if (this.count() == 0) return {};

    return {
        languages: {
            list: _.map(this.languages, function(lang) {
                return {
                    id: lang.id,
                    title: lang.title
                };
            })
        }
    };
};

module.exports = Langs;
