# Extend Filters

Filters are essentially functions that can be applied to variables. They are called with a pipe operator (`|`) and can take arguments.

```
{{ foo | title }}
{{ foo | join(",") }}
{{ foo | replace("foo", "bar") | capitalize }}
```

### Defining a new filter

Plugins can extend filters by defining custom functions in their entry point under the `filters` scope.

A filter function takes as first argument the content to filter, and should return the new content.
Refer to [Context and APIs](./api.md) to learn more about `this` and GitBook API.

```js
module.exports = {
    filters: {
        hello: function(name) {
            return 'Hello '+name;
        }
    }
};
```

The filter `hello` can then be used in the book:

```
{{ "Aaron"|hello }}, how are you?
```

### Handling block arguments

Arguments can be passed to filters:

```
Hello {{ "Samy"|fullName("Pesse", man=true}} }}
```

Arguments are passed to the function, named-arguments are passed as a last argument (object).

```js
module.exports = {
    filters: {
        fullName: function(firstName, lastName, kwargs) {
            var name = firstName + ' ' + lastName;

            if (kwargs.man) name = "Mr" + name;
            else name = "Mrs" + name;

            return name;
        }
    }
};
```