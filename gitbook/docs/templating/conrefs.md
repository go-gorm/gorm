# Content References

Content referencing (conref) is a convenient mechanism to reuse content from other files or books.

### Importing local files

Importing an other file's content is easy using the `include` tag:

```
{% include "./test.md" %}
```

### Importing file from another book

GitBook can also resolve the include path by using git:

```
{% include "git+https://github.com/GitbookIO/documentation.git/README.md#0.0.1" %}
```

The format of git url is:

```
git+https://user@hostname/owner/project.git/file#commit-ish
```

The real git url part should finish with `.git`, the filename to import is extracted after the `.git` till the fragment of the url.

The `commit-ish` can be any tag, sha, or branch which can be supplied as an argument to `git checkout`. The default is `master`.

### Inheritance

Template inheritance is a way to make it easy to reuse templates. When writing a template, you can define "blocks" that child templates can override. The inheritance chain can be as long as you like.

`block` defines a section on the template and identifies it with a name. Base templates can specify blocks and child templates can override them with new content.

```
{% extends "./mypage.md" %}

{% block pageContent %}
# This is my page content
{% endblock %}
```

In the file `mypage.md`, you should specify the blocks that can be extended:

```
{% block pageContent %}
This is the default content
{% endblock %}

# License

{% import "./LICENSE" %}
```
