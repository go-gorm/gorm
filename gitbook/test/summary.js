var should = require('should');

var mock = require('./mock');

function mockSummary(files, summary) {
    return mock.setupDefaultBook(files, summary)
    .then(function(book) {
        return book.readme.load()
        .then(function() {
            return book.summary.load();
        })
        .thenResolve(book);
    });
}

describe('Summary / Table of contents', function() {
    describe('Empty summary list', function() {
        var book;

        before(function() {
            return mockSummary({})
            .then(function(_book) {
                book = _book;
            });
        });

        it('should add README as first entry', function() {
            should(book.summary.getArticle('README.md')).be.ok();
        });

        it('should correctly count articles', function() {
            book.summary.count().should.equal(1);
        });
    });

    describe('Non-existant summary', function() {
        var book;

        before(function() {
            return mock.setupBook({
                'README.md': 'Hello'
            })
            .then(function(_book) {
                book = _book;

                return book.readme.load()
                .then(function() {
                    return book.summary.load();
                });
            });
        });

        it('should add README as first entry', function() {
            should(book.summary.getArticle('README.md')).be.ok();
        });

        it('should correctly count articles', function() {
            book.summary.count().should.equal(1);
        });
    });

    describe('Non-empty summary list', function() {
        var book;

        before(function() {
            return mockSummary({
                'SUMMARY.md': '# Summary\n\n'
                    + '* [Hello](./hello.md)\n'
                    + '* [World](./world.md)\n\n'
            })
            .then(function(_book) {
                book = _book;
            });
        });

        it('should correctly count articles', function() {
            book.summary.count().should.equal(3);
        });
    });

    describe('Levels', function() {
        var book;

        before(function() {
            return mockSummary({
                'SUMMARY.md': '# Summary\n\n'
                    + '* [Hello](./hello.md)\n'
                    + '    * [Hello 2](./hello2.md)\n'
                    + '* [World](./world.md)\n\n'
                    + '## Part 2\n\n'
                    + '* [Hello 3](./hello.md)\n'
                    + '    * [Hello 4](./hello2.md)\n'
            })
            .then(function(_book) {
                book = _book;
            });
        });

        it('should correctly index levels', function() {
            book.summary.getArticleByLevel('0').title.should.equal('Introduction');
            book.summary.getArticleByLevel('1.1').title.should.equal('Hello');
            book.summary.getArticleByLevel('1.1.1').title.should.equal('Hello 2');
            book.summary.getArticleByLevel('1.2').title.should.equal('World');

            book.summary.getArticleByLevel('2.1').title.should.equal('Hello 3');
            book.summary.getArticleByLevel('2.1.1').title.should.equal('Hello 4');
        });

        it('should correctly calcul depth', function() {
            book.summary.getArticleByLevel('0').depth().should.equal(1);
            book.summary.getArticleByLevel('1.1').depth().should.equal(2);
            book.summary.getArticleByLevel('1.1.1').depth().should.equal(3);
        });
    });

    describe('External', function() {
        var book;

        before(function() {
            return mockSummary({}, [
                {
                    title: 'Google',
                    path: 'https://www.google.fr'
                }
            ])
            .then(function(_book) {
                book = _book;
            });
        });

        it('should correctly count articles', function() {
            book.summary.count().should.equal(2);
        });

        it('should correctly signal it as external', function() {
            var article = book.summary.getArticleByLevel('1');

            should(article).be.ok();
            should(article.path).not.be.ok();

            article.title.should.equal('Google');
            article.ref.should.equal('https://www.google.fr');
            article.isExternal().should.be.ok;
        });
    });

    describe('Next / Previous', function() {
        var book;

        before(function() {
            return mockSummary({
                'SUMMARY.md': '# Summary\n\n' +
                    '* [Hello](hello.md)\n' +
                    '* [Hello 2](hello2.md)\n' +
                    '    * [Hello 3](hello3.md)\n' +
                    '    * [Hello 4](hello4.md)\n' +
                    '    * [Hello 5](hello5.md)\n' +
                    '* [Hello 6](hello6.md)\n\n\n' +
                    '### Part 2\n\n' +
                    '* [Hello 7](hello7.md)\n' +
                    '    * [Hello 8](hello8.md)\n\n' +
                    '### Part 3\n\n' +
                    '* [Hello 9](hello9.md)\n' +
                    '* [Hello 10](hello10.md)\n\n'
            })
            .then(function(_book) {
                book = _book;
            });
        });

        it('should only return a next for the readme', function() {
            var article = book.summary.getArticle('README.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).not.be.ok();
            should(next).be.ok();

            next.path.should.equal('hello.md');
        });

        it('should return next/prev for a first level page', function() {
            var article = book.summary.getArticle('hello.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('README.md');
            next.path.should.equal('hello2.md');
        });

        it('should return next/prev for a joint -> child', function() {
            var article = book.summary.getArticle('hello2.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello.md');
            next.path.should.equal('hello3.md');
        });

        it('should return next/prev for a joint <- child', function() {
            var article = book.summary.getArticle('hello3.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello2.md');
            next.path.should.equal('hello4.md');
        });

        it('should return next/prev for a children', function() {
            var article = book.summary.getArticle('hello4.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello3.md');
            next.path.should.equal('hello5.md');
        });

        it('should return next/prev for a joint -> parent', function() {
            var article = book.summary.getArticle('hello5.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello4.md');
            next.path.should.equal('hello6.md');
        });

        it('should return next/prev for a joint -> parts', function() {
            var article = book.summary.getArticle('hello6.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello5.md');
            next.path.should.equal('hello7.md');
        });

        it('should return next/prev for a joint <- parts', function() {
            var article = book.summary.getArticle('hello7.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello6.md');
            next.path.should.equal('hello8.md');
        });

        it('should return next and prev', function() {
            var article = book.summary.getArticle('hello8.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.ok();

            prev.path.should.equal('hello7.md');
            next.path.should.equal('hello9.md');
        });

        it('should return only prev for last', function() {
            var article = book.summary.getArticle('hello10.md');

            var prev = article.prev();
            var next = article.next();

            should(prev).be.ok();
            should(next).be.not.ok();

            prev.path.should.equal('hello9.md');
        });
    });
});

