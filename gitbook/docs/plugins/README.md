# Plugins

Plugins are the best way to extend GitBook functionalities (ebook and website). There exist plugins to do a lot of things: bring math formulas display support, track visits using Google Analytic, etc.

### How to find plugins?

Plugins can be easily searched on [plugins.gitbook.com](https://plugins.gitbook.com).


### How to install a plugin?

Once you find a plugin that you want to install, you need to add it to your `book.json`:

```
{
    "plugins": ["myPlugin", "anotherPlugin"]
}
```

You can also specify a specific version using: `"myPlugin@0.3.1"`. By default GitBook will resolve the latest version of the plugin compatbile with the current GitBook version.

### GitBook.com

Plugins are automatically installed on [GitBook.com](https://www.gitbook.com). Locally, run `gitbook install` to install and prepare all plugins for your books.

### Configuring plugins

Plugins specific configurations are stored in `pluginsConfig`. You have to refer to the documentation of the plugin itself for details about the available options.
