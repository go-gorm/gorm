# Context and APIs

GitBooks provides different APIs and contexts to plugins. These APIs can vary according to the GitBook version being used, your plugin should specify the `engines.gitbook` field in `package.json` accordingly.

#### Book instance

The `Book` class is the central point of GitBook, it centralize all access read methods. This class is defined in [book.js](https://github.com/GitbookIO/gitbook/blob/master/lib/book.js).

```js
// Read configuration from book.json
var value = book.config.get('title', 'Default Value');

// Resolve a filename to an absolute path
var filepath = book.resolve('README.md');

// Render an inline markup string
book.renderInline('markdown', 'This is **Markdown**')
    .then(function(str) { ... })

// Render a markup string (block mode)
book.renderBlock('markdown', '* This is **Markdown**')
    .then(function(str) { ... })
```

#### Output instance

The `Output` class represent the output/write process.

```js
// Return root folder for the output
var root = output.root();

// Resolve a file in the output folder
var filepath = output.resolve('myimage.png');

// Convert a filename to an URL (returns a path to an html file)
var fileurl = output.toURL('mychapter/README.md');

// Write a file in the output folder
output.write('hello.txt', 'Hello World')
    .then(function() { ... });

// Copy a file to the output folder
output.copyFile('./myfile.jpg', 'cover.jpg')
    .then(function() { ... });

// Verify that a file exists
output.hasFile('hello.txt')
    .then(function(exists) { ... });
```

#### Page instance

A page instance represent the current parsed page.

```js
// Title of the page (from SUMMARY)
page.title

// Content of the page (Markdown/Asciidoc/HTML according to the stage)
page.content

// Relative path in the book
page.path

// Absolute path to the file
page.rawPath

// Type of parser used for this file
page.type ('markdown' or 'asciidoc')
```

#### Context for Blocks and Filters

Blocks and filters have access to the same context, this context is bind to the template engine execution:

```js
{
    // Current templating syntax
    "ctx": {
        // For example, after a {% set message = "hello" %}
        "message": "hello"
    },

    // Book instance
    "book" <Book>,

    // Output instance
    "output": <Output>
}
```

For example a filter or block function can access the current book using: `this.book`.

#### Context for Hooks

Hooks only have access to the `<Book>` instance using `this.book`.
