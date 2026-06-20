package epub

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func getTestEpubPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), ".", "testdata", "test.epub")
}

func TestOpen(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	if epub == nil {
		t.Error("Expected EPUB object, got nil")
	}
}

func TestEpub_GetTitle(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	title := epub.GetTitle()
	println(title)
}

func TestEpub_GetAuthor(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	author := epub.GetAuthor()
	println(author)
}

func TestEpub_GetDescription(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	description := epub.GetDescription()
	println(description)
}

func TestEpub_GetChapters(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	chapters, err := epub.GetChapters()
	if err != nil {
		t.Fatalf("Failed to get chapters: %v", err)
	}

	if chapters[0].Order != 1 {
		t.Errorf("Expected first chapter order to be 1, got %d", chapters[0].Order)
	}

	if chapters[1].Order != 2 {
		t.Errorf("Expected second chapter order to be 2, got %d", chapters[1].Order)
	}
}

func TestEpub_GetChapterContent(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	content, err := epub.GetChapterContent(0)
	if err != nil {
		t.Fatalf("Failed to get chapter content: %v", err)
	}

	if content == "" {
		t.Error("Expected chapter content, got empty string")
	}

	// Test invalid chapter index
	_, err = epub.GetChapterContent(10000)
	if err == nil {
		t.Error("Expected error for invalid chapter index, got nil")
	}
}

func TestEpub_GetChapterReader(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	reader, err := epub.GetChapterReader(0)
	if err != nil {
		t.Fatalf("Failed to get chapter reader: %v", err)
	}

	if reader == nil {
		t.Error("Expected reader, got nil")
	}

	// Read content from reader
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from chapter reader: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected content from reader, got empty")
	}

	// Test invalid chapter index
	_, err = epub.GetChapterReader(10000)
	if err == nil {
		t.Error("Expected error for invalid chapter index, got nil")
	}
}

func TestEpub_GetFileReader(t *testing.T) {
	epub, err := Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer epub.Close()

	// Try to get a file we know exists
	reader, err := epub.GetFileReader("META-INF/container.xml")
	if err != nil {
		t.Fatalf("Failed to get file reader: %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Error("Expected reader, got nil")
	}

	// Read content from reader
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from file reader: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected content from reader, got empty")
	}

	// Check that it contains expected XML
	if !strings.Contains(string(content), "container") {
		t.Error("Expected container.xml content to contain 'container'")
	}

	// Test invalid file path
	_, err = epub.GetFileReader("nonexistent.xml")
	if err == nil {
		t.Error("Expected error for invalid file path, got nil")
	}
}

func TestNewReader(t *testing.T) {
	// Open test EPUB file as regular file
	file, err := os.Open(getTestEpubPath())
	if err != nil {
		t.Fatalf("Failed to open test EPUB file: %v", err)
	}
	defer file.Close()

	// Parse EPUB using NewReader
	epub, err := NewReader(file)
	if err != nil {
		t.Fatalf("Failed to parse EPUB with NewReader: %v", err)
	}

	if epub == nil {
		t.Fatal("Expected EPUB object, got nil")
	}

	// Verify basic metadata
	title := epub.GetTitle()
	if title == "" {
		t.Error("Expected title to be non-empty")
	}

	author := epub.GetAuthor()
	if author == "" {
		t.Error("Expected author to be non-empty")
	}

	// Verify we can get chapters
	chapters, err := epub.GetChapters()
	if err != nil {
		t.Fatalf("Failed to get chapters: %v", err)
	}

	if len(chapters) == 0 {
		t.Error("Expected to get at least one chapter")
	}

	// Verify we can get chapter content
	content, err := epub.GetChapterContent(0)
	if err != nil {
		t.Fatalf("Failed to get chapter content: %v", err)
	}

	if content == "" {
		t.Error("Expected chapter content to be non-empty")
	}
}
