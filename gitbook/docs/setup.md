# Setup and Installation of GitBook

Getting GitBook installed and ready-to-go should only take a few minutes.

### GitBook.com

[GitBook.com](https://www.gitbook.com) is an easy to use solution to write, publish and host books. It's best and easier solution for publishing your content and collaborate on it.

It integrates well with the [GitBook Editor](https://www.gitbook.com/editor).

### Local Installation

##### Requirements

Installing GitBook is easy and straight-forward, but there are a few requirements you’ll need to make sure your system has before you start.

* NodeJS (v4.0.0 and above are adviced)
* Windows, Linux, Unix, or Mac OS X

##### Install with NPM

The best way to install GitBook is via **NPM**. At the terminal prompt, simply run the following command to install GitBook:

```
$ npm install gitbook-cli -g
```

`gitbook-cli` is an utility to install and use multiple versions of GitBook on the same system. It will automatically install the required version to build a book.

##### Create a book

GitBook can setup a boilerplate book:

```
$ gitbook init
```

If you wish to create the book into a new directory, you can do so by running `gitbook init ./directory`

Preview and serve your book using:

```
$ gitbook serve
```

or build the static website using:

```
$ gitbook build
```

##### Install pre-releases

`gitbook-cli` makes it easy to install and test other versions of GitBook with your book:

```
$ gitbook fetch beta
```

Use `gitbook ls-remote` to list remote versions available for install.

##### Debugging

You can use the options `--log=debug` and `--debug` to get better error messages (with stack trace). For example:

```
$ gitbook build ./ --log=debug --debug
```

