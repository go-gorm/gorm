var _ = require('lodash');
var TypedError = require('error/typed');
var WrappedError = require('error/wrapped');
var deprecated = require('deprecated');

var Logger = require('./logger');

var log = new Logger();

// Enforce as an Error object, and cleanup message
function enforce(err) {
    if (_.isString(err)) err = new Error(err);
    err.message = err.message.replace(/^Error: /, '');

    return err;
}

// Random error wrappers during parsing/generation
var ParsingError = WrappedError({
    message: 'Parsing Error: {origMessage}',
    type: 'parse'
});
var OutputError = WrappedError({
    message: 'Output Error: {origMessage}',
    type: 'generate'
});

// A file does not exists
var FileNotFoundError = TypedError({
    type: 'file.not-found',
    message: 'No "{filename}" file (or is ignored)',
    filename: null
});

// A file is outside the scope
var FileOutOfScopeError = TypedError({
    type: 'file.out-of-scope',
    message: '"{filename}" not in "{root}"',
    filename: null,
    root: null,
    code: 'EACCESS'
});

// A file is outside the scope
var RequireInstallError = TypedError({
    type: 'install.required',
    message: '"{cmd}" is not installed.\n{install}',
    cmd: null,
    code: 'ENOENT',
    install: ''
});

// Error for nunjucks templates
var TemplateError = WrappedError({
    message: 'Error compiling template "{filename}": {origMessage}',
    type: 'template',
    filename: null
});

// Error for nunjucks templates
var PluginError = WrappedError({
    message: 'Error with plugin "{plugin}": {origMessage}',
    type: 'plugin',
    plugin: null
});

// Error with the book's configuration
var ConfigurationError = WrappedError({
    message: 'Error with book\'s configuration: {origMessage}',
    type: 'configuration'
});

// Error during ebook generation
var EbookError = WrappedError({
    message: 'Error during ebook generation: {origMessage}\n{stdout}',
    type: 'ebook',
    stdout: ''
});

// Deprecate methods/fields
function deprecateMethod(fn, msg) {
    return deprecated.method(msg, log.warn.ln, fn);
}
function deprecateField(obj, prop, value, msg) {
    return deprecated.field(msg, log.warn.ln, obj, prop, value);
}

module.exports = {
    enforce: enforce,

    ParsingError: ParsingError,
    OutputError: OutputError,
    RequireInstallError: RequireInstallError,

    FileNotFoundError: FileNotFoundError,
    FileOutOfScopeError: FileOutOfScopeError,

    TemplateError: TemplateError,
    PluginError: PluginError,
    ConfigurationError: ConfigurationError,
    EbookError: EbookError,

    deprecateMethod: deprecateMethod,
    deprecateField: deprecateField
};
