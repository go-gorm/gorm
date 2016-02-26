var Book = require('./book');
var cli = require('./cli');

module.exports = {
    Book: Book,
    commands: cli.commands
};
