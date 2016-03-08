# Extend Blocks

Extending templating blocks is the best way to provide extra functionalities to authors.

The most common usage is to process the content within some tags at runtime. It's like [filters](./filters.md), but on steroids because you aren't confined to a single expression.

### Defining a new block

Blocks are defined by the plugin, blocks is a map of name associated with a block descriptor. The block descriptor needs to contain at least a `process` method.

```js
module.exports = {
    blocks: {
        tag1: {
            process: function(block) {
                return "Hello "+block.body+", How are you?";
            }
        }
    }
};
```

The `process` should return the html content that will replace the tag. Refer to [Context and APIs](./api.md) to learn more about `this` and GitBook API.

### Handling block arguments

Arguments can be passed to blocks:

```
{% tag1 "argument 1", "argument 2", name="Test" %}
This is the body of the block.
{% endtag1 %}
```

And arguments are easily accessible in the `process` method:

```js
module.exports = {
    blocks: {
        tag1: {
            process: function(block) {
                // block.args equals ["argument 1", "argument 2"]
                // block.kwargs equals { "name": "Test" }
            }
        }
    }
};
```

### Handling sub-blocks

A defined block can be parsed into different sub-blocks, for example let's consider the source:

```
{% myTag %}
    Main body
    {% subblock1 %}
    Body of sub-block 1
    {% subblock 2 %}
    Body of sub-block 1
{% endmyTag %}
```
