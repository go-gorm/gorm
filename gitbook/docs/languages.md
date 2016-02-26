# Multi-Languages

GitBook supports building books written in multiple languages. Each language should be a sub-directory following the normal GitBook format, and a file named `LANGS.md` should be present at the root of the repository with the following format:

```markdown
* [English](en/)
* [French](fr/)
* [Español](es/)
```

### Configuration for each language

When a language book (ex: `en`) has a `book.json`, its configuration will extend the main configuration.

The only exception is plugins, plugins are specify globally relative to the book, and language specific plugins can not be specified.
