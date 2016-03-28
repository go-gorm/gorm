# Generating eBooks and PDFs

GitBook can generates a website, but can also output content as ebook (ePub, Mobi, PDF).

### Installing ebook-convert

`ebook-convert` is required to generate ebooks (epub, mobi, pdf).

##### OS X

Download the [Calibre application](https://calibre-ebook.com/download). After moving the `calibre.app` to your Applications folder create a symbolic link to the ebook-convert tool:

```
$ sudo ln -s ~/Applications/calibre.app/Contents/MacOS/ebook-convert /usr/bin
```

You can replace `/usr/bin` with any directory that is in your $PATH.

### Cover

Covers are used for all the ebook formats. It's an important part of an ebook brandline.

A good cover should respect the following guidelines:

* Size of 1800x2360 (pixels)
* No border
* Clearly visible book title
* Any important text should be visible in the small version

