var _ = require('lodash');
var util = require('util');
var BackboneFile = require('./file');

// Normalize a glossary entry name into a unique id
function nameToId(name) {
    return name.toLowerCase()
        .replace(/[\/\\\?\%\*\:\;\|\"\'\\<\\>\#\$\(\)\!\.\@]/g, '')
        .replace(/ /g, '_')
        .trim();
}


/*
A glossary entry is represented by a name and a short description
An unique id for the entry is generated using its name
*/
function GlossaryEntry(name, description) {
    if (!(this instanceof GlossaryEntry)) return new GlossaryEntry(name, description);

    this.name = name;
    this.description = description;

    Object.defineProperty(this, 'id', {
        get: _.bind(this.getId, this)
    });
}

// Normalizes a glossary entry's name to create an ID
GlossaryEntry.prototype.getId = function() {
    return nameToId(this.name);
};


/*
A glossary is a list of entries stored in a GLOSSARY.md file
*/
function Glossary() {
    BackboneFile.apply(this, arguments);

    this.entries = [];
}
util.inherits(Glossary, BackboneFile);

Glossary.prototype.type = 'glossary';

// Get templating context
Glossary.prototype.getContext = function() {
    if (!this.path) return {};

    return {
        glossary: {
            path: this.path
        }
    };
};

// Parse the readme content
Glossary.prototype.parse = function(content) {
    var that = this;

    return this.parser.glossary(content)
    .then(function(entries) {
        that.entries = _.map(entries, function(entry) {
            return new GlossaryEntry(entry.name, entry.description);
        });
    });
};

// Return an entry by its id
Glossary.prototype.get = function(id) {
    return _.find(this.entries, {
        id: id
    });
};

// Find an entry by its name
Glossary.prototype.find = function(name) {
    return this.get(nameToId(name));
};

// Return false if glossary has entries (and exists)
Glossary.prototype.isEmpty = function(id) {
    return _.size(this.entries) === 0;
};

// Convert the glossary to a list of annotations
Glossary.prototype.annotations = function() {
    return _.map(this.entries, function(entry) {
        return {
            id: entry.id,
            name: entry.name,
            description: entry.description,
            href: '/' + this.path + '#' + entry.id
        };
    }, this);
};

module.exports = Glossary;
