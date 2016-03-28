# Builtin Templating Helpers

GitBook provides a serie of builtin filters and blocks to help you write templates.

### Filters

`value|default(default, [boolean])`If value is strictly undefined, return default, otherwise value. If boolean is true, any JavaScript falsy value will return default (false, "", etc)

`arr|sort(reverse, caseSens, attr)`
Sort arr with JavaScript's arr.sort function. If reverse is true, result will be reversed. Sort is case-insensitive by default, but setting caseSens to true makes it case-sensitive. If attr is passed, will compare attr from each item.

### Blocks

`{% markdown %}Markdown string{% endmarkdown %}`
Render inline markdown

`{% asciidoc %}AsciiDoc string{% endasciidoc %}`
Render inline asciidoc
