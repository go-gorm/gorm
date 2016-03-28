var fs = require('fs');

var mock = require('./mock');
var WebsiteOutput = require('../lib/output/website');

describe('Website Output', function() {

    describe('Sample Book', function() {
        var output;

        before(function() {
            return mock.outputDefaultBook(WebsiteOutput)
            .then(function(_output) {
                output = _output;
            });
        });

        it('should correctly generate an index.html', function() {
            output.should.have.file('index.html');
        });

        it('should correctly copy assets', function() {
            output.should.have.file('gitbook/app.js');
            output.should.have.file('gitbook/images/favicon.ico');
        });

        it('should correctly copy plugins', function() {
            output.should.have.file('gitbook/gitbook-plugin-highlight/website.css');
        });
    });

    describe('Book with chapters', function() {
        var output;

        before(function() {
            return mock.outputDefaultBook(WebsiteOutput, {
                'hello/README.md': '# Hello',
                'hello/test.md': '# Test'
            }, [
                {
                    title: 'Hello',
                    path: 'hello/README.md'
                },
                {
                    title: 'Test',
                    path: 'hello/test.md'
                }
            ])
            .then(function(_output) {
                output = _output;
            });
        });

        it('should correctly generate an index.html', function() {
            output.should.have.file('index.html');
        });

        it('should correctly generate files in folder', function() {
            output.should.have.file('hello/index.html');
            output.should.have.file('hello/test.html');
        });
    });

    describe('Multilingual Book', function() {
        var output;

        before(function() {
            return mock.outputBook(WebsiteOutput, {
                'LANGS.md': '# Languages\n\n'
                    + '* [en](./en)\n'
                    + '* [fr](./fr)\n\n',
                'en/README.md': '# Hello',
                'fr/README.md': '# Bonjour'

            })
            .then(function(_output) {
                output = _output;
            });
        });

        it('should correctly generate an index.html for each language', function() {
            output.should.have.file('en/index.html');
            output.should.have.file('fr/index.html');
        });

        it('should correctly copy assets', function() {
            output.should.have.file('gitbook/app.js');
        });

        it('should not copy assets for each language', function() {
            output.should.have.not.file('en/gitbook/app.js');
            output.should.have.not.file('fr/gitbook/app.js');
        });

        it('should correctly generate an index.html', function() {
            output.should.have.file('index.html');
        });
    });

    describe('Theming', function() {
        var output;

        before(function() {
            return mock.outputDefaultBook(WebsiteOutput, {
                '_layouts/website/page.html': '{% extends "website/page.html" %}{% block body %}{{ super() }}<div id="theming-added"></div>{% endblock %}'

            })
            .then(function(_output) {
                output = _output;
            });
        });

        it('should extend default theme', function() {
            var readme = fs.readFileSync(output.resolve('index.html'), 'utf-8');

            readme.should.be.html({
                '#theming-added': {
                    count: 1
                }
            });
        });
    });
});

