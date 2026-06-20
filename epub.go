// Package epub provides functionality for reading and parsing EPUB files.
//
// It allows you to open EPUB files and extract their metadata, chapters, and
// other content. The package provides both high-level functions for common
// operations and low-level access to the internal structure of EPUB files.
//
// Basic usage:
//
//	package main
//
//	import (
//		"fmt"
//		"log"
//		"io"
//		"os"
//
//		"github.com/setcreed/go-epub/epub"
//	)
//
//	func main() {
//		// Open an EPUB file
//		e, err := epub.Open("book.epub")
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer e.Close()
//
//		// Get book metadata
//		fmt.Println("Title:", e.GetTitle())
//		fmt.Println("Author:", e.GetAuthor())
//		fmt.Println("Description:", e.GetDescription())
//
//		// Get chapters
//		chapters, err := e.GetChapters()
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		fmt.Printf("Found %d chapters\n", len(chapters))
//
//		// Read content of the first chapter using io.Reader
//		reader, err := e.GetChapterReader(0)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		// Copy the content to stdout
//		_, err = io.Copy(os.Stdout, reader)
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Epub represents an EPUB file
//
// The Epub struct contains the parsed contents of an EPUB file,
// including its metadata, manifest, spine, and table of contents.
// It also maintains a reference to the underlying zip.Reader for
// accessing the raw file contents.
type Epub struct {
	File     *zip.Reader
	RootFile string
	Metadata Metadata
	Manifest []Item
	Spine    []ItemRef
	TOC      *NCX

	// Store the ReadCloser for closing when needed
	readCloser io.Closer
}

// Metadata represents the metadata of an EPUB
type Metadata struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Subject     string `xml:"subject"`
	Description string `xml:"description"`
	Publisher   string `xml:"publisher"`
	Contributor string `xml:"contributor"`
	Date        string `xml:"date"`
	Type        string `xml:"type"`
	Format      string `xml:"format"`
	Identifier  string `xml:"identifier"`
	Language    string `xml:"language"`
	Rights      string `xml:"rights"`
}

// Container represents the container.xml file structure
type Container struct {
	Rootfiles []Rootfile `xml:"rootfiles>rootfile"`
}

// Rootfile represents root file information
type Rootfile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// Package represents the package document structure
type Package struct {
	Metadata Metadata  `xml:"metadata"`
	Manifest []Item    `xml:"manifest>item"`
	Spine    []ItemRef `xml:"spine>itemref"`
}

// Item represents an item in the manifest
type Item struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// ItemRef represents an item reference in the spine
type ItemRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

// NCX represents the NCX file structure (table of contents)
type NCX struct {
	Title  string     `xml:"docTitle>text"`
	NavMap []NavPoint `xml:"navMap>navPoint"`
}

// NavPoint represents a navigation point (chapter)
type NavPoint struct {
	ID        string     `xml:"id,attr"`
	PlayOrder string     `xml:"playOrder,attr"`
	Label     string     `xml:"navLabel>text"`
	Content   string     `xml:"content"`
	Src       string     `xml:"content,attr"`
	NavPoints []NavPoint `xml:"navPoint"`
}

// Chapter represents a book chapter
//
// A Chapter contains the title, content, and order of a chapter
// in the EPUB file. Chapters are extracted based on the spine
// order defined in the EPUB package document.
type Chapter struct {
	Title   string
	Content string
	Order   int
}

// Open opens and parses an EPUB file from a file path
//
// The Open function takes a path to an EPUB file and returns a pointer to an
// Epub struct that represents the parsed contents of the file. It handles all
// the necessary parsing of the EPUB structure, including the container file,
// package document, and table of contents.
//
// It is the caller's responsibility to call Close on the returned Epub when
// finished with it to free up resources.
//
// Example:
//
//	e, err := epub.Open("book.epub")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer e.Close()
//
//	title := e.GetTitle()
func Open(path string) (*Epub, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	epub := &Epub{
		File:       &reader.Reader,
		readCloser: reader,
	}

	if err := epub.parseContainer(); err != nil {
		epub.Close()
		return nil, err
	}

	if err := epub.parsePackage(); err != nil {
		epub.Close()
		return nil, err
	}

	if err := epub.parseTOC(); err != nil {
		epub.Close()
		return nil, err
	}

	return epub, nil
}

// New creates and parses an EPUB from a zip.Reader
//
// The New function takes a zip.Reader and returns a pointer to an
// Epub struct that represents the parsed contents of the EPUB.
// This is useful when you already have a zip.Reader and want to parse
// it as an EPUB file.
//
// Example:
//
//	// When you have an io.Reader containing EPUB data
//	reader, err := os.Open("book.epub")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer reader.Close()
//
//	// Get file info to create a sized reader
//	stat, err := reader.Stat()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a zip reader
//	zipReader, err := zip.NewReader(reader, stat.Size())
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Parse as EPUB
//	e, err := epub.New(zipReader)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer e.Close()
func New(r *zip.Reader) (*Epub, error) {
	epub := &Epub{
		File: r,
	}

	if err := epub.parseContainer(); err != nil {
		return nil, err
	}

	if err := epub.parsePackage(); err != nil {
		return nil, err
	}

	if err := epub.parseTOC(); err != nil {
		return nil, err
	}

	return epub, nil
}

// NewReader creates and parses an EPUB from an io.Reader
//
// The NewReader function takes an io.Reader and returns a pointer to an
// Epub struct that represents the parsed contents of the EPUB.
// This is useful when you have an io.Reader and want to parse it as an EPUB file.
// Note that this function reads the entire content into memory to provide random access.
//
// Example:
//
//	// When you have an io.Reader containing EPUB data
//	reader, err := os.Open("book.epub")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer reader.Close()
//
//	// Parse as EPUB
//	e, err := epub.NewReader(reader)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer e.Close()
func NewReader(r io.Reader) (*Epub, error) {
	// Read all data into memory to provide random access
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Create a reader that implements ReaderAt
	readerAt := bytes.NewReader(data)

	// Create a zip reader
	zipReader, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return nil, err
	}

	// Create epub
	epub := &Epub{
		File: zipReader,
	}

	if err := epub.parseContainer(); err != nil {
		return nil, err
	}

	if err := epub.parsePackage(); err != nil {
		return nil, err
	}

	if err := epub.parseTOC(); err != nil {
		return nil, err
	}

	return epub, nil
}

// parseContainer parses the META-INF/container.xml file
func (e *Epub) parseContainer() error {
	containerFile, err := e.getFile("META-INF/container.xml")
	if err != nil {
		return err
	}

	var container Container
	if err := xml.Unmarshal(containerFile, &container); err != nil {
		return err
	}

	if len(container.Rootfiles) > 0 {
		e.RootFile = container.Rootfiles[0].FullPath
	}

	return nil
}

// parsePackage parses the package document (.opf file)
func (e *Epub) parsePackage() error {
	packageFile, err := e.getFile(e.RootFile)
	if err != nil {
		return err
	}

	var pkg Package
	if err := xml.Unmarshal(packageFile, &pkg); err != nil {
		return err
	}

	e.Metadata = pkg.Metadata
	e.Manifest = pkg.Manifest
	e.Spine = pkg.Spine

	return nil
}

// parseTOC parses the NCX table of contents file
func (e *Epub) parseTOC() error {
	// Try to find the NCX file first (EPUB 2.0)
	ncxItem := e.findItemByMediaType("application/x-dtbncx+xml")
	if ncxItem != nil {
		// Get NCX file content
		ncxPath := filepath.Join(filepath.Dir(e.RootFile), ncxItem.Href)
		ncxData, err := e.getFile(ncxPath)
		if err != nil {
			return err
		}

		// Parse NCX
		var ncx NCX
		if err := xml.Unmarshal(ncxData, &ncx); err != nil {
			return err
		}

		e.TOC = &ncx
		return nil
	}

	// For EPUB 3.0, try to find the navigation document
	// Look for item with properties="nav"
	for _, item := range e.Manifest {
		// In EPUB 3, the navigation document is identified by properties="nav"
		// We'll need to extend the Item struct to support this, but for now
		// we can look for HTML files that might be the navigation document
		if strings.Contains(item.MediaType, "html") {
			// Try to parse as navigation document - simplified approach
			// A full implementation would parse the HTML nav structure
			continue
		}
	}

	// If no TOC found, that's okay - not all EPUBs have a traditional TOC
	return nil
}

// getFile gets the content of a file from the EPUB by path
func (e *Epub) getFile(path string) ([]byte, error) {
	path = filepath.ToSlash(path)

	for _, file := range e.File.File {
		if filepath.ToSlash(file.Name) == path {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("file not found: %s", path)
}

// findItemByID finds an item in the manifest by ID
func (e *Epub) findItemByID(id string) *Item {
	for _, item := range e.Manifest {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

// findItemByMediaType finds an item in the manifest by media type
func (e *Epub) findItemByMediaType(mediaType string) *Item {
	for _, item := range e.Manifest {
		if item.MediaType == mediaType {
			return &item
		}
	}
	return nil
}

// findItemByHref finds an item in the manifest by href
func (e *Epub) findItemByHref(href string) *Item {
	href = filepath.ToSlash(href)
	for _, item := range e.Manifest {
		if filepath.ToSlash(item.Href) == href {
			return &item
		}
	}
	return nil
}

// GetTitle returns the book title
//
// This method returns the title of the EPUB book as defined in its metadata.
// If no title is defined in the EPUB metadata, an empty string is returned.
func (e *Epub) GetTitle() string {
	return e.Metadata.Title
}

// GetAuthor returns the book author
//
// This method returns the creator/author of the EPUB book as defined in its metadata.
// If no author is defined in the EPUB metadata, an empty string is returned.
func (e *Epub) GetAuthor() string {
	return e.Metadata.Creator
}

// GetDescription returns the book description
//
// This method returns the description of the EPUB book as defined in its metadata.
// If no description is defined in the EPUB metadata, an empty string is returned.
func (e *Epub) GetDescription() string {
	return e.Metadata.Description
}

// GetMetadata returns the complete metadata of the book
//
// This method returns the complete metadata struct of the EPUB book,
// which includes all available metadata fields like title, author, subject,
// description, publisher, etc.
//
// Example:
//
//	metadata := e.GetMetadata()
//	fmt.Println("Title:", metadata.Title)
//	fmt.Println("Author:", metadata.Creator)
//	fmt.Println("Publisher:", metadata.Publisher)
func (e *Epub) GetMetadata() Metadata {
	return e.Metadata
}

// GetItems returns all items in the EPUB manifest
//
// This method returns the complete list of items declared in the EPUB manifest.
// Each item contains its ID, href (path), and media type. This can be useful
// for examining all resources included in the EPUB file.
//
// Example:
//
//	items := e.GetItems()
//	for _, item := range items {
//		fmt.Printf("ID: %s, Href: %s, MediaType: %s\n", item.ID, item.Href, item.MediaType)
//	}
func (e *Epub) GetItems() []Item {
	return e.Manifest
}

// GetCover returns a reader for the cover image of the EPUB, if one exists
//
// This method attempts to locate and return a reader for the cover image of the EPUB.
// Not all EPUBs have a cover image, and the location of the cover can vary between
// EPUB versions. If a cover image is found, an io.ReadCloser is returned which the
// caller must close. If no cover is found, nil is returned with no error.
//
// Example:
//
//	cover, err := e.GetCover()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if cover != nil {
//		defer cover.Close()
//		// Process cover image
//	} else {
//		fmt.Println("No cover image found")
//	}
func (e *Epub) GetCover() (io.ReadCloser, error) {
	// Try to find cover by meta tag (EPUB 2.0 and 3.0 method)
	// This would require parsing the metadata more thoroughly

	// Try common cover item IDs
	coverIDs := []string{"cover", "cover-image", "cover-img"}
	for _, id := range coverIDs {
		item := e.findItemByID(id)
		if item != nil && (strings.HasPrefix(item.MediaType, "image/") ||
			strings.HasSuffix(strings.ToLower(item.Href), ".jpg") ||
			strings.HasSuffix(strings.ToLower(item.Href), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(item.Href), ".png") ||
			strings.HasSuffix(strings.ToLower(item.Href), ".gif")) {

			return e.GetFileReader(item.Href)
		}
	}

	// If no cover found, return nil without error
	return nil, nil
}

// GetChapters returns all chapter content
//
// This method extracts all chapters from the EPUB file based on the spine order
// defined in the package document. It only processes items with HTML media types
// and attempts to extract chapter titles from the table of contents.
//
// The method returns a slice of Chapter structs containing the title, content,
// and order of each chapter. If there are no chapters or an error occurs during
// processing, an empty slice and an error may be returned.
//
// Example:
//
//	chapters, err := e.GetChapters()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, chapter := range chapters {
//		fmt.Printf("Chapter %d: %s\n", chapter.Order, chapter.Title)
//	}
func (e *Epub) GetChapters(opts ...Option) ([]Chapter, error) {
	options := applyOptions(opts...)

	// Check if context is already cancelled
	if err := options.checkContext(); err != nil {
		return nil, err
	}

	var chapters []Chapter

	// Get chapters according to spine order
	for i, itemRef := range e.Spine {
		// Check for cancellation periodically
		if i%5 == 0 && options.isCancelled() {
			return nil, options.ctx.Err()
		}

		item := e.findItemByID(itemRef.IDRef)
		if item == nil {
			continue
		}

		// Only process HTML content files
		if strings.Contains(item.MediaType, "html") {
			chapterPath := filepath.Join(filepath.Dir(e.RootFile), item.Href)
			content, err := e.getFile(chapterPath)
			if err != nil {
				continue
			}

			// Apply content length filter if set
			if options.MaxContentLength > 0 && int64(len(content)) > options.MaxContentLength {
				continue
			}

			// Extract chapter title (may need more complex parsing)
			title := fmt.Sprintf("Chapter %d", i+1)
			if e.TOC != nil && i < len(e.TOC.NavMap) {
				title = e.TOC.NavMap[i].Label
			}

			chapter := Chapter{
				Title:   title,
				Content: string(content),
				Order:   i + 1,
			}

			// Apply chapter filter if set
			if options.FilterChapters != nil && !options.FilterChapters(chapter) {
				continue
			}

			chapters = append(chapters, chapter)
		}
	}

	return chapters, nil
}

// GetChapterContent returns the content of a specific chapter
//
// This method returns the content of a chapter at the specified index as a string.
// The index is zero-based, so the first chapter is at index 0.
//
// If the chapter index is out of range or an error occurs while retrieving the
// chapter content, an error is returned.
//
// Example:
//
//	content, err := e.GetChapterContent(0)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(content)
func (e *Epub) GetChapterContent(chapterIndex int, opts ...Option) (string, error) {
	options := applyOptions(opts...)

	// Check if context is already cancelled
	if err := options.checkContext(); err != nil {
		return "", err
	}

	// Validate chapter index by checking spine
	if chapterIndex < 0 || chapterIndex >= len(e.Spine) {
		return "", fmt.Errorf("chapter index out of range")
	}

	itemRef := e.Spine[chapterIndex]
	item := e.findItemByID(itemRef.IDRef)
	if item == nil {
		return "", fmt.Errorf("chapter item not found")
	}

	// Only process HTML content files
	if !strings.Contains(item.MediaType, "html") {
		return "", fmt.Errorf("chapter is not an HTML document")
	}

	chapterPath := filepath.Join(filepath.Dir(e.RootFile), item.Href)
	content, err := e.getFile(chapterPath)
	if err != nil {
		return "", fmt.Errorf("failed to get chapter content: %w", err)
	}

	// Apply content length filter if set
	if options.MaxContentLength > 0 && int64(len(content)) > options.MaxContentLength {
		return "", fmt.Errorf("chapter content exceeds maximum length")
	}

	return string(content), nil
}

// GetChapterReader returns an io.Reader for a specific chapter
//
// This method returns an io.Reader for the content of a chapter at the specified index.
// The index is zero-based, so the first chapter is at index 0.
//
// This is useful when you want to stream the chapter content rather than load it
// entirely into memory. The returned reader can be used with standard Go io operations.
//
// If the chapter index is out of range or an error occurs while retrieving the
// chapter content, an error is returned.
//
// Example:
//
//	reader, err := e.GetChapterReader(0)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Copy the content to stdout
//	_, err = io.Copy(os.Stdout, reader)
//	if err != nil {
//		log.Fatal(err)
//	}
func (e *Epub) GetChapterReader(chapterIndex int, opts ...Option) (io.Reader, error) {
	content, err := e.GetChapterContent(chapterIndex, opts...)
	if err != nil {
		return nil, err
	}

	return strings.NewReader(content), nil
}

// GetFileReader returns an io.Reader for a file in the EPUB by path
//
// This method returns an io.ReadCloser for any file within the EPUB archive,
// identified by its path. The path should be relative to the root of the EPUB.
//
// This is useful for accessing specific files within the EPUB, such as CSS files,
// images, or other resources. The caller is responsible for closing the returned
// ReadCloser when finished with it.
//
// If the specified file is not found in the EPUB, an error is returned.
//
// Example:
//
//	reader, err := e.GetFileReader("META-INF/container.xml")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer reader.Close()
//
//	content, err := io.ReadAll(reader)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(string(content))
func (e *Epub) GetFileReader(path string) (io.ReadCloser, error) {
	path = filepath.ToSlash(path)

	for _, file := range e.File.File {
		if filepath.ToSlash(file.Name) == path {
			return file.Open()
		}
	}

	return nil, fmt.Errorf("file not found: %s", path)
}

// Close closes the EPUB file
//
// This method closes the underlying EPUB file and releases any associated resources.
// It should be called when finished working with the EPUB to prevent resource leaks.
//
// Example:
//
//	e, err := epub.Open("book.epub")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer e.Close() // Ensures the file is closed when done
func (e *Epub) Close() error {
	if e.readCloser != nil {
		return e.readCloser.Close()
	}
	return nil
}
