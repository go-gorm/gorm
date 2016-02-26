var mock = require('./mock');
var Output = require('../lib/output/base');

describe('Page', function() {
    var book, output;

    before(function() {
        return mock.setupDefaultBook({
            'README.md': ' # Hello World\n\nThis is a description',
            'heading.md': '# Hello\n\n## World',
            'description.md': '# This is a title\n\nThis is the short description.\n\nNot this one.',
            'frontmatter.md': '---\ndescription: Hello World\n---\n\n# This is a title\n\nThis is not the description',

            'links.md': '[link](hello.md) [link 2](variables/page/next.md) [readme](README.md)',
            'links/relative.md': '[link](../hello.md) [link 2](/variables/page/next.md) [readme](../README.md)',

            'images.md': '![this is an image](test.png) ![this is an absolute image](/test2.png) ![this is a remote image](https://upload.wikimedia.org/wikipedia/commons/4/47/PNG_transparency_demonstration_1.png)',
            'images/relative.md': '![this is an image](test.png) ![this is an absolute image](/test2.png)',

            'annotations/simple.md': 'A magicien say abracadabra!',
            'annotations/code.md': 'A magicien say `abracadabra`!',
            'annotations/class.md': 'A magicien say <div class="no-glossary"><b>abracadabra</b>, right?</div>!',

            'codes/simple.md': '```hello world```',
            'codes/lang.md': '```js\nhello world\n```',
            'codes/lang.adoc': '```js\nhello world\n```',

            'folder/paths.md': '',

            'variables/file/mtime.md': '{{ file.mtime }}',
            'variables/file/path.md': '{{ file.path }}',
            'variables/page/title.md': '{{ page.title }}',
            'variables/page/previous.md': '{{ page.previous.title }} {{ page.previous.path }}',
            'variables/page/next.md': '{{ page.next.title }} {{ page.next.path }}',
            'variables/page/dir/ltr.md': 'This is english: {{ page.dir }}',
            'variables/page/dir/rtl.md': 'بسيطة {{ page.dir }}',
            'variables/config/title.md': '{{ config.title}}',

            'GLOSSARY.md': '# Glossary\n\n\n## abracadabra\n\nthis is the description'
        }, [
            {
                title: 'Test page.next',
                path: 'variables/page/next.md'
            },
            {
                title: 'Test Variables',
                path: 'variables/page/title.md'
            },
            {
                title: 'Test page.previous',
                path: 'variables/page/previous.md'
            }
        ])
        .then(function(_book) {
            book = _book;
            output = new Output(book);

            return book.parse();
        });
    });

    describe('.resolveLocal', function() {
        it('should correctly resolve path to file', function() {
            var page = book.addPage('heading.md');

            page.resolveLocal('test.png').should.equal('test.png');
            page.resolveLocal('/test.png').should.equal('test.png');
            page.resolveLocal('test/hello.png').should.equal('test/hello.png');
            page.resolveLocal('/test/hello.png').should.equal('test/hello.png');
        });

        it('should correctly resolve path to file (2)', function() {
            var page = book.addPage('folder/paths.md');

            page.resolveLocal('test.png').should.equal('folder/test.png');
            page.resolveLocal('/test.png').should.equal('test.png');
            page.resolveLocal('test/hello.png').should.equal('folder/test/hello.png');
            page.resolveLocal('/test/hello.png').should.equal('test/hello.png');
        });
    });

    describe('.relative', function() {
        it('should correctly resolve absolute path in the book', function() {
            var page = book.addPage('heading.md');
            page.relative('/test.png').should.equal('test.png');
            page.relative('test.png').should.equal('test.png');

            var page2 = book.addPage('folder/paths.md');
            page2.relative('/test.png').should.equal('../test.png');
            page2.relative('test.png').should.equal('../test.png');
        });
    });

    describe('.resolve', function() {
        var page;

        before(function() {
            page = book.addPage('links/relative.md');
        });

        it('should resolve to a relative path (same folder)', function() {
            page.relative('links/test.md').should.equal('test.md');
        });

        it('should resolve to a relative path (parent folder)', function() {
            page.relative('test.md').should.equal('../test.md');
            page.relative('hello/test.md').should.equal('../hello/test.md');
        });

        it('should resolve to a relative path (child folder)', function() {
            page.relative('links/hello/test.md').should.equal('hello/test.md');
        });
    });

    describe('Headings', function() {
        it('should add a default ID to headings', function() {
            var page = book.addPage('heading.md');

            return page.toHTML(output)
            .then(function() {
                page.content.should.be.html({
                    'h1#hello': {
                        count: 1
                    },
                    'h2#world': {
                        count: 1
                    }
                });
            });
        });
    });

    describe('Description', function() {
        it('should extratc page description from content', function() {
            var page = book.addPage('description.md');

            return page.toHTML(output)
            .then(function() {
                page.description.should.equal('This is the short description.');
            });
        });
    });

    describe('Font-Matter', function() {
        it('should extratc page description from front matter', function() {
            var page = book.addPage('frontmatter.md');

            return page.toHTML(output)
            .then(function() {
                page.description.should.equal('Hello World');
            });
        });
    });

    describe('Code Blocks', function() {
        var page;

        before(function() {
            output.template.addBlock('code', function(blk) {
                return (blk.kwargs.language || '') + blk.body + 'test';
            });
        });

        it('should apply "code" block', function() {
            page = book.addPage('codes/simple.md');
            return page.toHTML(output)
                .should.be.fulfilledWith('<p><code>hello worldtest</code></p>\n');
        });

        it('should add language as kwargs', function() {
            page = book.addPage('codes/lang.md');
            return page.toHTML(output)
                .should.be.fulfilledWith('<pre><code class="lang-js">jshello world\ntest</code></pre>\n');
        });

        it('should add language as kwargs (asciidoc)', function() {
            page = book.addPage('codes/lang.adoc');
            return page.toHTML(output)
                .should.be.fulfilledWith('<div class="listingblock">\n<div class="content">\n<pre class="highlight"><code class="language-js" data-lang="js">jshello worldtest</code></pre>\n</div>\n</div>');
        });
    });

    describe('Links', function() {
        describe('From base directory', function() {
            var page;

            before(function() {
                page = book.addPage('links.md');
                return page.toHTML(output);
            });

            it('should replace links to page to .html', function() {
                page.content.should.be.html({
                    'a[href="variables/page/next.html"]': {
                        count: 1
                    }
                });
            });

            it('should use directory urls when file is a README', function() {
                page.content.should.be.html({
                    'a[href="./"]': {
                        count: 1
                    }
                });
            });

            it('should not replace links to file not in SUMMARY', function() {
                page.content.should.be.html({
                    'a[href="hello.md"]': {
                        count: 1
                    }
                });
            });
        });

        describe('From sub-directory', function() {
            var page;

            before(function() {
                page = book.addPage('links/relative.md');
                return page.toHTML(output);
            });

            it('should replace links to page to .html', function() {
                page.content.should.be.html({
                    'a[href="../variables/page/next.html"]': {
                        count: 1
                    }
                });
            });

            it('should use directory urls when file is a README', function() {
                page.content.should.be.html({
                    'a[href="../"]': {
                        count: 1
                    }
                });
            });

            it('should not replace links to file not in SUMMARY', function() {
                page.content.should.be.html({
                    'a[href="../hello.md"]': {
                        count: 1
                    }
                });
            });
        });
    });

    describe('Images', function() {
        describe('From base directory', function() {
            var page;

            before(function() {
                page = book.addPage('images.md');
                return page.toHTML(output);
            });

            it('should resolve relative images', function() {
                page.content.should.be.html({
                    'img[src="test.png"]': {
                        count: 1
                    }
                });
            });

            it('should resolve absolute images', function() {
                page.content.should.be.html({
                    'img[src="test2.png"]': {
                        count: 1
                    }
                });
            });

            it('should keep external images path', function() {
                page.content.should.be.html({
                    'img[src="https:/upload.wikimedia.org/wikipedia/commons/4/47/PNG_transparency_demonstration_1.png"]': {
                        count: 1
                    }
                });
            });
        });

        describe('From sub-directory', function() {
            var page;

            before(function() {
                page = book.addPage('images/relative.md');
                return page.toHTML(output);
            });

            it('should resolve relative images', function() {
                page.content.should.be.html({
                    'img[src="test.png"]': {
                        count: 1
                    }
                });
            });

            it('should resolve absolute images', function() {
                page.content.should.be.html({
                    'img[src="../test2.png"]': {
                        count: 1
                    }
                });
            });
        });
    });

    describe('Templating Context', function() {
        it('should set file.mtime', function() {
            var page = book.addPage('variables/file/mtime.md');
            return page.toHTML(output)
            .then(function(content) {
                // A date ends with "(CET)" or "(UTC)""
                content.should.endWith(')</p>\n');
            });
        });

        it('should set file.path', function() {
            var page = book.addPage('variables/file/path.md');
            return page.toHTML(output)
            .should.be.fulfilledWith('<p>variables/file/path.md</p>\n');
        });

        it('should set page.title when page is in summary', function() {
            var page = book.getPage('variables/page/title.md');
            return page.toHTML(output)
            .should.be.fulfilledWith('<p>Test Variables</p>\n');
        });

        it('should set page.previous when possible', function() {
            var page = book.getPage('variables/page/previous.md');
            return page.toHTML(output)
            .should.be.fulfilledWith('<p>Test Variables variables/page/title.md</p>\n');
        });

        it('should set page.next when possible', function() {
            var page = book.getPage('variables/page/next.md');
            return page.toHTML(output)
            .should.be.fulfilledWith('<p>Test Variables variables/page/title.md</p>\n');
        });

        it('should set config.title', function() {
            var page = book.addPage('variables/config/title.md');
            return page.toHTML(output)
            .should.be.fulfilledWith('<p>Hello World</p>\n');
        });

        describe('page.dir', function() {
            it('should detect ltr', function() {
                var page = book.addPage('variables/page/dir/ltr.md');
                return page.toHTML(output)
                .should.be.fulfilledWith('<p>This is english: ltr</p>\n');
            });

            it('should detect rtl', function() {
                var page = book.addPage('variables/page/dir/rtl.md');
                return page.toHTML(output)
                .should.be.fulfilledWith('<p>&#x628;&#x633;&#x64A;&#x637;&#x629; rtl</p>\n');
            });
        });
    });

    describe('Annotations / Glossary', function() {
        it('should replace glossary terms', function() {
            return book.addPage('annotations/simple.md').toHTML(output)
            .should.finally.be.html({
                '.glossary-term': {
                    count: 1,
                    text: 'abracadabra',
                    attributes: {
                        title: 'this is the description',
                        href: '../GLOSSARY.html#abracadabra'
                    }
                }
            });
        });

        it('should not replace terms in code blocks', function() {
            return book.addPage('annotations/code.md').toHTML(output)
            .should.finally.be.html({
                '.glossary-term': {
                    count: 0
                }
            });
        });

        it('should not replace terms in ".no-glossary"', function() {
            return book.addPage('annotations/class.md').toHTML(output)
            .should.finally.be.html({
                '.glossary-term': {
                    count: 0
                }
            });
        });
    });
});
