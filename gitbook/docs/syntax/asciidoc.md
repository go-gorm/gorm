# AsciiDoc

Since version `2.0.0`, GitBook can also accept AsciiDoc as an input format.

Please refer to the [AsciiDoc Syntax Quick Reference](http://asciidoctor.org/docs/asciidoc-syntax-quick-reference/) for more informations about the format.

Just like for markdown, GitBook is using some special files to extract structures: `README.adoc`, `SUMMARY.adoc`, `LANGS.adoc` and `GLOSSARY.adoc`.

### README.adoc

This is the main entry of your book: the introduction. This file is **required**.

### SUMMARY.adoc

This file defines the list of chapters and subchapters. Just like  in Markdown, the `SUMMARY.adoc`'s format is simply a list of links, the name of the link is used as the chapter's name, and the target is a path to that chapter's file.

Subchapters are defined simply by adding a nested list to a parent chapter.

```asciidoc
= Summary

. link:chapter-1/README.adoc[Chapter 1]
.. link:chapter-1/ARTICLE1.adoc[Article 1]
.. link:chapter-1/ARTICLE2.adoc[Article 2]
... link:chapter-1/ARTICLE-1-2-1.adoc[Article 1.2.1]
. link:chapter-2/README.adoc[Chapter 2]
. link:chapter-3/README.adoc[Chapter 3]
. link:chapter-4/README.adoc[Chapter 4]
.. Unfinished article
. Unfinished Chapter
```

### LANGS.adoc

For [Multi-Languages](./languages.md) books, this file is used to define the different supported languages and translations.

This file is following the same syntax as the `SUMMARY.adoc`:

```asciidoc
= Languages

. link:en/[English]
. link:fr/[French]
```

### GLOSSARY.adoc

This file is used to define terms. [See the glossary section](./lexicon.md).

```asciidoc
= Glossary

== Magic

Sufficiently advanced technology, beyond the understanding of the
observer producing a sense of wonder.

== PHP

A popular web programming language, used by many large websites such
as Facebook. Rasmus Lerdorf originally created PHP in 1994 to power
his personal homepage (PHP originally stood for "Personal Home Page"
but now stands for "PHP: Hypertext Preprocessor"). ```


