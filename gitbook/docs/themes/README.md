# Theming

Since version 3.0.0, GitBook can be easily themed. Books are using by default the [theme-default](https://github.com/GitbookIO/theme-default).

> **Caution**: Custom theming can block some plugins from working correctly.

### Structure of a theme

A theme is a plugin containing templates and assets. All the templates are optionnal, since theme are always extending the default theme.

| Folder | Description |
| -------- | ----------- |
| `_layouts` | Main folder containing all the templates |
| `_layouts/website/page.html` | Template for a normal page |
| `_layouts/ebook/page.html` | Template for a normal page during ebook generation (PDF< ePub, Mobi) |


### Extend/Customize theme in a book

Authors can extend the templates of a theme directly from the book source (without creating an external theme). Templates will be resolved in the `_layouts` folder of the book first, then in

### Publish a theme

Themes are published as plugins ([see related docs](../plugins/README.md)) with a `theme-` prefix. For example the theme `awesome` will be loaded from the `theme-awesome` plugin, and then from the `gitbook-plugin-theme-awesome` NPM package.
