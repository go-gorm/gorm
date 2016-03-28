# Markdown

Most of the examples from this documentation are in Markdown. Markdown is default parser for GitBook, but one can also opt for the [AsciiDoc syntax](asciidoc.md).

Here’s an overview of Markdown syntax that you can use with GitBook (same as GitHub with some additions).

### Headings

To create a heading, add one to six `#` symbols before your heading text. The number of # you use will determine the size of the heading.

```markdown
# This is an <h1> tag
## This is an <h2> tag
###### This is an <h6> tag
```

GitBook supports a nice way for explicitly setting the header ID. If you follow the header text with an opening curly bracket (separated from the text with a least one space), a hash, the ID and a closing curly bracket, the ID is set on the header. If you use the trailing hash feature of atx style headers, the header ID has to go after the trailing hashes. For example:

```markdown
Hello {#id}
-----

# Hello {#id}

# Hello # {#id}
```

### Paragraphs and Line Breaks {#paragraphs}

A paragraph is simply one or more consecutive lines of text, separated by one or more blank lines. (A blank line is any line that looks like a blank line — a line containing nothing but spaces or tabs is considered blank.) Normal paragraphs should not be indented with spaces or tabs.

```
Here's a line for us to start with.

This line is separated from the one above by two newlines, so it will be a *separate paragraph*.
```

### Emphasis {#emphasis}

```markdown
*This text will be italic*
_This will also be italic_

**This text will be bold**
__This will also be bold__

~~This text will be crossed out.~~

_You **can** combine them_
```

### Lists {#lists}

Markdown supports ordered (numbered) and unordered (bulleted) lists.

##### Unordered

Unordered lists use asterisks, pluses, and hyphens — interchangably — as list markers:

```markdown
* Item 1
* Item 2
  * Item 2a
  * Item 2b
```

##### Ordered

Ordered lists use numbers followed by periods:

```markdown
1. Item 1
2. Item 2
3. Item 3
   * Item 3a
   * Item 3b
```

### Links {#links}

Markdown supports two style of links: inline and reference.

A simple link can be created by surrounding the text with square brackets and the link URL with parentheses:

```markdown
This is [an example](http://example.com/ "Title") inline link with a title.

[This link](http://example.net/) has no title attribute.
```

Links can point to relative paths, anchors or absolute urls.


### References

There is another way to create links which does not interrupt the text flow. The URL and title are defined using a reference name and this reference name is then used in square brackets instead of the link URL:

```markdown
This is [an example][id] reference-style link.
```

Then, anywhere in the document, you define your link label like this, on a line by itself:

```markdown
[id]: http://example.com/  "Optional Title Here"
```

### Images {#images}

Images can be created in a similar way than links: just use an exclamation mark before the square brackets. The link text will become the alternative text of the image and the link URL specifies the image source:

```markdown
An image: ![gras](img/image.jpg)
```

### Blockquotes {#blockquotes}

A blockquote is started using the `>` marker followed by an optional space; all following lines that are also started with the blockquote marker belong to the blockquote. You can use any block-level elements inside a blockquote:

```markdown
As Kanye West said:

> We're living the future so
> the present is our past.
```

### Tables {#tables}

You can create tables by assembling a list of words and dividing them with hyphens `-` (for the first row), and then separating each column with a pipe `|`:

```markdown
| First Header  | Second Header |
| ------------- | ------------- |
| Content Cell  | Content Cell  |
| Content Cell  | Content Cell  |
```

The pipes on either end of the table are optional. Cells can vary in width and do not need to be perfectly aligned within columns. There must be at least three hyphens in each column of the header row.

### Code {#code}

Markdown supports two different code block styles. One uses lines indented with either four spaces or one tab whereas the other uses lines with tilde characters as delimiters – therefore the content does not need to be indented:

```markdown
This is a sample code block.

    Continued here.

```

##### Fenced code blocks

You can create fenced code blocks by placing triple backticks ` ``` ` before and after the code block. We recommend placing a blank line before and after code blocks to make the raw formatting easier to read.

    ```
    function test() {
      console.log("notice the blank line before this function?");
    }
    ```

##### Syntax highlighting

You can add an optional language identifier to enable syntax highlighting in your fenced code block.

For example, to syntax highlight Ruby code:

    ```ruby
    require 'redcarpet'
    markdown = Redcarpet.new("Hello World!")
    puts markdown.to_html
    ```

##### Inline code

Text phrases can be marked up as code by surrounding them with backticks:

    Use `gitbook` to convert the `text` in markdown
    syntax to HTML.

### Footnotes

GitBook supports a simple syntax for such footnotes. Footnotes are relative to each pages.

```markdown
Text prior to footnote reference.[^2]

[^2]: Comment to include in footnote.
```

### HTML

GitBook supports use of raw HTML in your text, Markdown syntax in HTML is not processed:

```
<div>
Markdown here will not be **parsed**
</div>
```

### Horizontal Rule

Horizontal Rules can be inserted using three or more asterisks, dashes or underscores, optionally separated by spaces or tabs, on an otherwise blank line:

```markdown
Three or more...

---

Hyphens

***

Asterisks

```

### Ignoring Markdown formatting

You can tell GitBook to ignore (or escape) Markdown formatting by using `\` before the Markdown character.

```
Let's rename \*our-new-project\* to \*our-old-project\*.
```
