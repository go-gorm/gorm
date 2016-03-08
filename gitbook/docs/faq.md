# GitBook FAQ

#### How can I host/publish my book?

Books can easily be published and hosted on [GitBook.com](https://www.gitbook.com). But GitBook output can be hosted on any static file hosting solution.

#### What can I use to edit my content?

Any text editor should work! But we advise using the [GitBook Editor](https://www.gitbook.com/editor). [GitBook.com](https://www.gitbook.com) also provides a web version of this editor.

---

#### Should I use an `.html` or `.md` extensions in my links?

You should always use paths and the `.md` extensions when linking to your files, GitBook will automatically replace these paths by the appropriate link when the pointing file is referenced in the Table of Contents.

#### Can I create a GitBook in a sub-directory of my repository?

Yes, GitBooks can be created in [sub-directories](structure.md#subdirectory). GitBook.com and the CLI also looks by default in a serie of [folders](structure.md).

#### Does GitBook supports RTL languages?

Yes, GitBook automatically detect the direction in your pages (`rtl` or `ltr`) and adjust the layout accordingly. The direction can also be specified globally in the [book.json](config.md).

---

#### Does GitBook support Math equations?

GitBook supports math equations and TeX thanks to plugins. There are currently 2 official plugins to display math: [mathjax](https://plugins.gitbook.com/plugin/mathjax) and [katex](https://plugins.gitbook.com/plugin/katex).

#### Can I customize/theme the output?

Yes, both the website and ebook outputs can be customized using [themes](themes/README.md).

#### Can I add interactive content (videos, etc)?

GitBook is very [extensible](plugins/README.md). You can use [existing plugins](https://plugins.gitbook.com) or create your own!
