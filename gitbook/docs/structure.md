# Directory structure

GitBook uses a very simple and obvious directory sttructure:

```
.
├── book.json
├── README.md
├── SUMMARY.md
├── chapter-1/
|   ├── README.md
|   └── something.md
└── chapter-2/
    ├── README.md
    └── something.md
```

An overview of what each of these does:

| File | Description |
| -------- | ----------- |
| `book.json` | Stores [configuration](config.md) data (__optional__) |
| `README.md` | Preface / Introduction for your book (**required**) |
| `SUMMARY.md` | Table of Contents |


### Static files and Images

A static file is a file that is not listed in the `SUMMARY.md`. All static files, not [ignored](#ignore), are copied to the output.

### Ignoring files & folders {#ignore}

GitBook will read the `.gitignore`, `.bookignore` and `.ignore` files to get a list of files and folders to skip.
The format inside those files, follows the same convention as `.gitignore`:

```markdown
# This is a comment

# Ignore the file test.md
test.md

# Ignore everything in the directory "bin"
bin/*
```

### Project documentation / Sub-directory {#subdirectory}

For project documentaiton, it sometimes better to use a diretcory (like `docs/`) to store the prject's documentation. You can use a `.gitbook` file to indicate to GitBook in which folder the book is stored:

```
.
├── .gitbook
└── docs/
    ├── README.md
    └── SUMMARY.md
```

With `.gitbook` containing:

```
./docs/
```
