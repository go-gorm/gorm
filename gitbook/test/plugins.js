var _ = require('lodash');
var path = require('path');

var mock = require('./mock');
var registry = require('../lib/plugins/registry');
var Output = require('../lib/output/base');
var PluginsManager = require('../lib/plugins');
var BookPlugin = require('../lib/plugins/plugin');

var PLUGINS_ROOT = path.resolve(__dirname, 'node_modules');

describe('Plugins', function() {
    var book;

    before(function() {
        return mock.setupBook({})
        .then(function(_book) {
            book = _book;
        });
    });

    describe('Resolve Version', function() {
        it('should resolve a plugin version', function() {
            return registry.resolve('ga')
            .should.be.fulfilled();
        });
    });

    describe('Installation', function() {
        it('should install a plugin from NPM without a specific version', function() {
            return registry.install(book, 'ga')
            .should.be.fulfilled();
        });

        it('should install a plugin from NPM with a specific version', function() {
            return registry.install(book, 'ga', '1.0.0')
            .should.be.fulfilled();
        });

        it('should correctly install all dependencies (if none)', function() {
            return mock.setupBook({})
            .then(function(book) {
                var plugins = new PluginsManager(book);
                return plugins.install()
                .should.be.fulfilledWith(0);
            });
        });

        it('should correctly install all dependencies (if any)', function() {
            return mock.setupBook({
                'book.json': {
                    plugins: ['ga']
                }
            })
            .then(function(book) {
                return book.prepareConfig()
                .then(function() {
                    var plugins = new PluginsManager(book);
                    return plugins.install();
                });
            })
            .should.be.fulfilledWith(1);
        });

        it('should correctly install dependencies from GitHub', function() {
            return mock.setupBook({
                'book.json': {
                    plugins: ['ga@git+https://github.com/GitbookIO/plugin-ga#master']
                }
            })
            .then(function(book) {
                return book.prepareConfig()
                .then(function() {
                    var plugins = new PluginsManager(book);
                    return plugins.install();
                });
            })
            .should.be.fulfilledWith(1);
        });
    });

    describe('Loading', function() {
        it('should load default plugins', function() {
            return mock.outputDefaultBook(Output)
            .then(function(output) {
                output.plugins.count().should.be.greaterThan(0);
            });
        });
    });

    describe('Configuration', function() {
        it('should fail loading a plugin with an invalid configuration', function() {
            var plugin = new BookPlugin(book, 'test-config');
            return plugin.load(PLUGINS_ROOT)
                .should.be.rejectedWith('Error with book\'s configuration: pluginsConfig.test-config.myProperty is required');
        });

        it('should extend configuration with default properties', function() {
            return mock.setupBook({
                'book.json': {
                    pluginsConfig: {
                        'test-config': {
                            'myProperty': 'world'
                        }
                    }
                }
            })
            .then(function(book2) {
                return book2.prepareConfig()
                .then(function() {
                    var plugin = new BookPlugin(book2, 'test-config');
                    return plugin.load(PLUGINS_ROOT);
                })
                .then(function() {
                    book2.config.get('pluginsConfig.test-config.myDefaultProperty', '').should.equal('hello');
                });
            });
        });
    });

    describe('Resources', function() {
        var plugin;

        before(function() {
            plugin = new BookPlugin(book, 'test-resources');
            return plugin.load(PLUGINS_ROOT);
        });


        it('should list all resources for website', function() {
            return plugin.getResources('website')
            .then(function(resources) {
                resources.assets.should.equal('./assets');

                resources.js.should.have.lengthOf(2);
                resources.js[0].path.should.equal('gitbook-plugin-test-resources/myfile.js');
                resources.js[1].url.should.equal('https://ajax.googleapis.com/ajax/libs/angularjs/1.4.9/angular.min.js');

                resources.css.should.have.lengthOf(1);
                resources.css[0].path.should.equal('gitbook-plugin-test-resources/myfile.css');
            });
        });
    });

    describe('Filters', function() {
        var plugin, filters;

        before(function() {
            plugin = new BookPlugin(book, 'test-filters');
            return plugin.load(PLUGINS_ROOT)

            .then(function() {
                filters = plugin.getFilters();
            });
        });

        it('should list all filters', function() {
            _.size(filters).should.equal(2);
        });

        it('should correctly execute a filter', function() {
            filters.hello('World').should.equal('Hello World!');
        });

        it('should correctly set contexts for filter', function() {
            filters.testContext('Hello');
        });
    });

    describe('Blocks', function() {
        var plugin, blocks;

        before(function() {
            plugin = new BookPlugin(book, 'test-blocks');
            return plugin.load(PLUGINS_ROOT)

            .then(function() {
                blocks = plugin.getBlocks();
            });
        });

        it('should list all blocks', function() {
            _.size(blocks).should.equal(2);
        });

        it('should correctly normalize block', function() {
            blocks.hello.process({ body: 'World' }).should.equal('Hello World!');
        });

        it('should correctly set contexts for filter', function() {
            blocks.testContext.process({ body: 'Hello' });
        });
    });

    describe('Hooks', function() {
        var plugin;

        before(function() {
            plugin = new BookPlugin(book, 'test-hooks');
            return plugin.load(PLUGINS_ROOT);
        });

        it('can call a hook', function() {
            return plugin.hook('init')
            .then(function() {
                global._hooks.should.deepEqual(['init']);
            });
        });
    });
});

