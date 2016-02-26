var mock = require('./mock');
var ConrefsLoader = require('../lib/output/conrefs')();


describe('Conrefs Loader', function() {
    var output;

    before(function() {
        return mock.outputDefaultBook(ConrefsLoader, {
            'test.md': 'World'
        })
        .then(function(_output) {
            output = _output;
        });
    });


    it('should include a local file', function() {
        return output.template.renderString('Hello {% include "./test.md" %}')
            .should.be.fulfilledWith('Hello World');
    });

    it('should include a git url', function() {
        return output.template.renderString('Hello {% include "./test.md" %}')
            .should.be.fulfilledWith('Hello World');
    });

    it('should reject file out of scope', function() {
        return output.template.renderString('Hello {% include "../test.md" %}')
            .should.be.rejected();
    });

    describe('Git Urls', function() {
        it('should include a file from a git repo', function() {
            return output.template.renderString('{% include "git+https://gist.github.com/69ea4542e4c8967d2fa7.git/test.md" %}')
                .should.be.fulfilledWith('Hello from git');
        });

        it('should handle deep inclusion (1)', function() {
            return output.template.renderString('{% include "git+https://gist.github.com/69ea4542e4c8967d2fa7.git/test2.md" %}')
                .should.be.fulfilledWith('First Hello. Hello from git');
        });

        it('should handle deep inclusion (2)', function() {
            return output.template.renderString('{% include "git+https://gist.github.com/69ea4542e4c8967d2fa7.git/test3.md" %}')
                .should.be.fulfilledWith('First Hello. Hello from git');
        });
    });
});
