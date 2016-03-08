# Create and publish a plugin

A GitBook plugin is a node package published on NPM that follow a defined convention.

## Structure

#### package.json

The `package.json` is a manifest format for describing **Node.js modules**. GitBook plugins are built on top of Node modules. It declares dependencies, version, ownership, and other information required to run a plugin in GitBook. This document describes the schema in detail.

A plugin manifest `package.json` can also contain details about the required configuration. The configuration schema is defined in the `gitbook` field of the `package.json` (This field follow the [JSON-Schema](http://json-schema.org) guidelines):

```js
{
    "name": "gitbook-plugin-mytest",
    "version": "0.0.1",
    "description": "This is my first GitBook plugin",
    "engines": {
        "gitbook": ">1.x.x"
    },
    "gitbook": {
        "properties": {
            "myConfigKey": {
                "type": "string",
                "default": "it's the default value",
                "description": "It defines my awesome config!"
            }
        }
    }
}
```

You can learn more about `package.json` from the [NPM documentation](https://docs.npmjs.com/files/package.json).

The **package name** must begin with `gitbook-plugin-` and the **package engines** should contains `gitbook`.

#### index.js

The `index.js` is main entry point of your plugin runtime:

```js
module.exports = {
    // Map of hooks
    hooks: {},

    // Map of new blocks
    blocks: {},

    // Map of new filters
    filters: {}
};
```

## Publish your plugin

GitBook plugins can be published on [NPM](https://www.npmjs.com).

To publish a new plugin, you need to create an account on [npmjs.com](https://www.npmjs.com) then publish it from the command line:

```
$ npm publish
```

## Private plugins

Private plugins can be hosted on GitHub and included using `git` urls:

```
{
    "plugins": [
        "myplugin@git+https://github.com/MyCompany/mygitbookplugin.git#1.0.0"
    ]
}
```
