# Pages and Summary

GitBook uses a `SUMMARY.md` file to define the structure of chapters and subchapters of the book. The `SUMMARY.md` file is used to generate the book's table of contents.

### Summary

The `SUMMARY.md`'s format is simply a list of links, the title of the link is used as the chapter's title, and the target is a path to that chapter's file.

Subchapters are defined simply by adding a nested list to a parent chapter.

##### Simple example

```markdown
# Summary

* [Part I](part1/README.md)
    * [Writing is nice](part1/writing.md)
    * [GitBook is nice](part1/gitbook.md)
* [Part II](part2/README.md)
    * [We love feedback](part2/feedback_please.md)
    * [Better tools for authors](part2/better_tools.md)
```

##### Example with subchapters split into parts

```markdown
# Summary

### Part 1

* [Writing is nice](part1/writing.md)
* [GitBook is nice](part1/gitbook.md)

### Part 2

* [We love feedback](part2/feedback_please.md)
* [Better tools for authors](part2/better_tools.md)
```

### Front Matter

Pages can contain an optional front matter. It can be used to define the page's description. The front matter must be the first thing in the file and must take the form of valid YAML set between triple-dashed lines. Here is a basic example:

```yaml
---
description: This is a short description of my page
---

# The content of my page
```
