# Hooks

Hooks is a method of augmenting or altering the behavior of the process, with custom callbacks.

### List of hooks

### Relative to the global pipeline

| Name | Description | Arguments |
| ---- | ----------- | --------- |
| `init` | Called after parsing the book, before generating output and pages. | None |
| `finish:before` | Called after generating the pages, before copying assets, cover, ... | None |
| `finish` | Called after everything else. | None |

### Relative to the page pipeline

> It is recommended using [templating](./templating.md) to extend page parsing.

| Name | Description | Arguments |
| ---- | ----------- | --------- |
| `page:before` | Called before running the templating engine on the page | Page Object |
| `page` | Called before outputting and indexing the page. | Page Object |

##### Page Object

```js
{
    // Parser named
    "type": "markdown",

    // File Path relative to book root
    "path": "page.md",

    // Absolute file path
    "rawpath": "/usr/...",

    // Title of the page in the SUMMARY
    "title": "",

    // Content of the page
    // Markdown/Asciidoc in "page:before"
    // HTML in "page"
    "content": "# Hello"
}
```

##### Example to add a title

In the `page:before` hook, `page.content` is the markdown/asciidoc content.

```js
{
    "page:before": function(page) {
        page.content = "# Title\n" +page.content;
        return page;
    }
}
```

##### Example to replace some html

In the `page` hook, `page.content` is the HTML generated from the markdown/asciidoc conversion.

```js
{
    "page": function(page) {
        page.content = page.content.replace("<b>", "<strong>")
            .replace("</b>", "</strong>");
        return page;
    }
}
```


### Asynchronous Operations

Hooks callbacks can be asynchronous and return promises.

Example:

```js
{
    "init": function() {
        return writeSomeFile()
        .then(function() {
            return writeAnotherFile();
        });
    }
}
```
