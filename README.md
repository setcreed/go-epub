# go-epub

A Go library for reading and parsing EPUB files.


## Features

- Read EPUB file metadata (title, author, description, etc.)
- Extract chapters and their content
- Access any file within an EPUB archive
- io.Reader interface support for reading content
- Simple and intuitive API
- Get cover image from EPUB
- Support for both EPUB 2.0 and 3.0 files
- Context support for cancellation and timeouts
- Option pattern for flexible configuration

## Installation

```bash
go get github.com/setcreed/go-epub
```


For documentation on specific functions:

```bash
go doc epub.Open
go doc epub.Epub.GetChapters
```

## Usage

### Basic Example

```go
package main

import (
	"fmt"
	"log"
	"io"
	"os"

	"github.com/setcreed/go-epub/epub"
)

func main() {
	// Open an EPUB file
	e, err := epub.Open("book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()

	// Get book metadata
	fmt.Println("Title:", e.GetTitle())
	fmt.Println("Author:", e.GetAuthor())
	fmt.Println("Description:", e.GetDescription())

	// Get chapters
	chapters, err := e.GetChapters()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d chapters\n", len(chapters))

	// Read content of the first chapter using io.Reader
	reader, err := e.GetChapterReader(0)
	if err != nil {
		log.Fatal(err)
	}

	// Copy the content to stdout
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Fatal(err)
	}
}
```

### Reading Files from EPUB

```go
// Get a reader for any file in the EPUB
reader, err := e.GetFileReader("META-INF/container.xml")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

content, err := io.ReadAll(reader)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(content))
```

### Working with Chapters

```go
// Get all chapters
chapters, err := e.GetChapters()
if err != nil {
    log.Fatal(err)
}

// Print information about each chapter
for _, chapter := range chapters {
    fmt.Printf("Chapter %d: %s\n", chapter.Order, chapter.Title)
    
    // Get chapter content as string
    content, err := e.GetChapterContent(chapter.Order - 1)
    if err != nil {
        log.Printf("Error reading chapter %d: %v", chapter.Order, err)
        continue
    }
    
    fmt.Printf("Content length: %d bytes\n", len(content))
}
```

### Using Context and Options

```go
import (
    "context"
    "time"
    "github.com/setcreed/go-epub/epub"
)

// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Get chapters with context and options
chapters, err := e.GetChapters(
    epub.WithContext(ctx),
    epub.WithChapterFilter(func(chapter epub.Chapter) bool {
        // Only include chapters with content longer than 100 characters
        return len(chapter.Content) > 100
    }),
    epub.WithMaxContentLength(1024*1024), // 1MB limit
)

if err != nil {
    log.Fatal(err)
}
```

### Getting the Cover Image

```go
// Try to get the cover image
cover, err := e.GetCover()
if err != nil {
    log.Fatal(err)
}

if cover != nil {
    defer cover.Close()
    // Process cover image (e.g. copy to file)
    out, err := os.Create("cover.jpg")
    if err != nil {
        log.Fatal(err)
    }
    defer out.Close()
    
    _, err = io.Copy(out, cover)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Cover image saved to cover.jpg")
} else {
    fmt.Println("No cover image found")
}
```


## API Reference

### `epub.Epub`

The main struct representing an EPUB file.

#### Methods

- `Open(path string) (*Epub, error)` - Open and parse an EPUB file
- `New(r *zip.Reader) (*Epub, error)` - Create EPUB from a zip.Reader
- `GetTitle() string` - Get the book title
- `GetAuthor() string` - Get the book author
- `GetDescription() string` - Get the book description
- `GetMetadata() Metadata` - Get complete book metadata
- `GetItems() []Item` - Get all items in the manifest
- `GetChapters(...Option) ([]Chapter, error)` - Get all chapters with options
- `GetChapterContent(chapterIndex int, ...Option) (string, error)` - Get content of a specific chapter as string
- `GetChapterReader(chapterIndex int, ...Option) (io.Reader, error)` - Get content of a specific chapter as io.Reader
- `GetFileReader(path string) (io.ReadCloser, error)` - Get a reader for any file in the EPUB
- `GetCover() (io.ReadCloser, error)` - Get the cover image of the EPUB
- `Close() error` - Close the EPUB file


### `epub.Chapter`

Represents a book chapter.

Fields:
- `Title string` - Chapter title
- `Content string` - Chapter content
- `Order int` - Chapter order

### `epub.Document`

Represents a parsed document from the EPUB file.

Fields:
- `Title string` - Document title
- `Content string` - Document content
- `MediaType string` - Document media type
- `ID string` - Document ID

### `epub.Metadata`

Represents the metadata of an EPUB.

Fields:
- `Title string` - The title of the book
- `Creator string` - The creator/author of the book
- `Subject string` - The subject of the book
- `Description string` - A description of the book
- `Publisher string` - The publisher of the book
- `Contributor string` - Additional contributors
- `Date string` - Publication date
- `Type string` - The type of the book
- `Format string` - The format of the book
- `Identifier string` - Unique identifier for the book
- `Language string` - Language of the book
- `Rights string` - Copyright information

### `epub.Item`

Represents an item in the manifest.

Fields:
- `ID string` - Unique identifier for the item
- `Href string` - Path to the item within the EPUB
- `MediaType string` - MIME type of the item

### Options

- `WithContext(ctx context.Context) Option` - Set context for cancellation and timeout
- `WithChapterFilter(filter func(chapter Chapter) bool) Option` - Filter chapters with a custom function
- `WithMaxContentLength(maxLen int64) Option` - Set maximum content length to process

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

MIT
