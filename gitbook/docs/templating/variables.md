# Variables

The following is a reference of the available data during book's parsing and theme generation.

### Global Variables

| Variable | Description |
| -------- | ----------- |
| `book` | Book-wide information + configuration settings from `book.json`. See below for details. |
| `gitbook` | GitBook specific information |
| `page` | Current page specific information |
| `file` | File associated with the current page specific information |
| `summary` | Information about the table of contents |
| `languages` | List of languages for multi-lingual books |
| `output` | Information about the output generator |
| `config` | Dump of the `book.json` |

### Book Variables

| Variable | Description |
| -------- | ----------- |
| `book.[CONFIGURATION_DATA]` | All the `variables` set via the `book.json` are available through the book variable. |
| `book.language` | Current language for a multilingual book |

### GitBook Variables

| Variable | Description |
| -------- | ----------- |
| `gitbook.time` | The current time (when you run the `gitbook` command) . |
| `gitbook.version` | Version of GitBook used to generate the book |

### File Variables

| Variable | Description |
| -------- | ----------- |
| `file.path` | The path to the raw page |
| `file.mtime` | Modified Time. Last time the file was modified |
| `file.type` | The name of the parser used to compile this file (ex: `markdown`, `asciidoc`, etc) |

#### Page Variables

| Variable | Description |
| -------- | ----------- |
| `page.title` | Title of the page |
| `page.previous` | Previous page in the Table of Contents (can be `null`) |
| `page.next` | Next page in the Table of Contents (can be `null`) |
| `page.dir` | Text direction, based on configuration or detected from content (`rtl` or `ltr`) |

#### Table of Contents Variables

| Variable | Description |
| -------- | ----------- |
| `summary.parts` | List of sections in the Table of Contents |

The whole table of contents (`SUMMARY.md`) can be accessed:

`summary.parts[0].articles[0].title` will return the title of the first article.

#### Multi-lingual book Variable

| Variable | Description |
| -------- | ----------- |
| `languages.list` | List of languages for this book |

Languages are defined by `{ id: 'en', title: 'English' }`.

### Output Variables

| Variable | Description |
| -------- | ----------- |
| `output.name` | Name of the output generator, possible values are `website`, `json`, `ebook` |
| `output.format` | When `output.name == "ebook"`, `format` defines the ebook format that will be generated, possible values are `pdf`, `epub` or `mobi` |

