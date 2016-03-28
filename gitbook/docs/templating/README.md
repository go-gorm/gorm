# Templating

GitBook uses the [Nunjucks templating language](https://mozilla.github.io/nunjucks/) to process pages and theme's templates.

The Nunjucks syntax is very similar to **Jinja2** or **Liquid**. Its syntax uses surrounding braces `{ }` to mark content that needs to be processed.

### Variables

A variable looks up a value from the template context. If you wanted to simply display a variable, you would use the `{{ variable }}` syntax. For example :

```twig
My name is {{ name }}, nice to meet you
```

This looks up username from the context and displays it. Variable names can have dots in them which lookup properties, just like JavaScript. You can also use the square bracket syntax.

```twig
{{ foo.bar }}
{{ foo["bar"] }}
```

If a value is undefined, nothing is displayed. The following all output nothing if foo is undefined: `{{ foo }}`, `{{ foo.bar }}`, `{{ foo.bar.baz }}`.

GitBook provides a set of [predefined  variables](variables.md) from the context.

### Filters

Filters are essentially functions that can be applied to variables. They are called with a pipe operator (`|`) and can take arguments.

```twig
{{ foo | title }}
{{ foo | join(",") }}
{{ foo | replace("foo", "bar") | capitalize }}
```

The third example shows how you can chain filters. It would display "Bar", by first replacing "foo" with "bar" and then capitalizing it.

### Tags

##### if

`if` tests a condition and lets you selectively display content. It behaves exactly as JavaScript's `if` behaves.

```twig
{% if variable %}
  It is true
{% endif %}
```

If variable is defined and evaluates to true, "It is true" will be displayed. Otherwise, nothing will be.

You can specify alternate conditions with `elif` and `else`:

```twig
{% if hungry %}
  I am hungry
{% elif tired %}
  I am tired
{% else %}
  I am good!
{% endif %}
```

##### for

`for` iterates over arrays and dictionaries.

```twig
# Chapters about GitBook

{% for article in glossary.terms['gitbook'].articles %}
* [{{ article.title }}]({{ article.path }})
{% endfor %}
```

##### set

`set` lets you create/modify a variable.

```twig
{% set softwareVersion = "1.0.0" %}

Current version is {{ softwareVersion }}.
[Download it](website.com/download/{{ softwareVersion }})
```

##### include and block

Inclusion and inheritance is detailled in the [Content References](conrefs.md) section.

### Escaping

If you want GitBook to ignore any of the special templating tags, you can use raw and anything inside of it will be output as plain text.

``` twig
{% raw %}
  this will {{ not be processed }}
{% endraw %}
```
