# Configuration

GitBook allows you to customize your book using a flexible configuration. These options are specified in a `book.json` file. For authors unfamiliar with the JSON syntax, you can validate the syntax using tools such as [JSONlint](http://jsonlint.com).

### General Settings

| Variable | Description |
| -------- | ----------- |
| `root` | Path to the root folder containing all the book's files, except `book.json`|
| `title` | Title of your book, default value is extracted from the README. On GitBook.com this field is pre-filled. |
| `description` | Description of your book, default value is extracted from the README. On GitBook.com this field is pre-filled. |
| `author` | Name of the author. On GitBook.com this field is pre-filled. |
| `isbn` | ISBN of the book |
| `language` | [ISO code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) of the book's language, default value is `en` |
| `direction` | Text's direction. Can be `rtl` or `ltr`, the default value depends on the value of `language` |
| `gitbook` | Version of GitBook that should be used. Uses the [SemVer](http://semver.org) specification and accepts conditions like `">= 3.0.0"` |

### Plugins

Plugins and their configurations are specified in the `book.json`. See [the plugins section](plugins/README.md) for more details.

| Variable | Description |
| -------- | ----------- |
| `plugins` | List of plugins to load |
| `pluginsConfig` |Configuration for plugins |

### Theme

Since version 3.0.0, GitBook can use themes. See [the theming section](themes/README.md) for more details.

| Variable | Description |
| -------- | ----------- |
| `theme` | The theme to use for the book |

### PDF Options

PDF Output can be customized using a set of options in the `book.json`:

| Variable | Description |
| -------- | ----------- |
| `pdf.pageNumbers` | Add page numbers to the bottom of every page (default is `true`) |
| `pdf.fontSize` | Base font size (default is `12`) |
| `pdf.fontFamily` | Base font family (default is `Arial`) |
| `pdf.paperSize` | Paper size, options are `'a0', 'a1', 'a2', 'a3', 'a4', 'a5', 'a6', 'b0', 'b1', 'b2', 'b3', 'b4', 'b5', 'b6', 'legal', 'letter'` (default is `a4`) |
| `pdf.margin.top` | Top margin (default is `56`) |
| `pdf.margin.bottom` | Bottom margin (default is `56`) |
| `pdf.margin.right` | Right margin (default is `62`) |
| `pdf.margin.left` | Left margin (default is `62`) |
