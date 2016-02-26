# Configuration

GitBook allows you to customize your book using a flexible configuration. These options are specified in a `book.json` file.

### Configuration Settings

| Variable | Description |
| -------- | ----------- |
| `title` | Title of your book, default value is extracted from the README. On GitBook.com this field is pre-filled. |
| `description` | Description of your book, default value is extracted from the README. On GitBook.com this field is pre-filled. |
| `author` | Name of the author. On GitBook.com this field is pre-filled. |
| `isbn` | ISBN of the book |
| `language` | ISO code of the book's language, default value is `en` |
| `direction` | `rtl` or `ltr`, default value depends on the value of `language` |
| `gitbook` | [SemVer](http://semver.org) condition to validate which GitBook version should be used |
| `plugins` | List of plugins to load, See [the plugins section](plugins.md) for more details |
| `pluginsConfig` |Configuration for plugins, See [the plugins section](plugins.md) for more details |

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

### Plugins

Plugins and their configurations are specified in the `book.json`. See [the plugins section](plugins.md) for more details.

