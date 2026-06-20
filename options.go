package epub

import (
	"context"
)

// Option defines a functional option for configuring EPUB parsing
type Option func(*epubOptions)

// epubOptions holds the configuration options for EPUB parsing
type epubOptions struct {
	// Context for cancellation and timeout support
	ctx context.Context
	
	// IncludeCover indicates whether to include cover images in the parsed content
	IncludeCover bool
	
	// IncludeMetadata indicates whether to include full metadata in the parsed content
	IncludeMetadata bool
	
	// FilterChapters allows filtering chapters by a custom function
	FilterChapters func(chapter Chapter) bool
	
	// MaxContentLength limits the maximum size of content to process
	MaxContentLength int64
}

// defaultOptions returns the default options
func defaultOptions() *epubOptions {
	return &epubOptions{
		ctx:              context.Background(),
		IncludeCover:     false,
		IncludeMetadata:  false,
		FilterChapters:   nil,
		MaxContentLength: 0, // No limit
	}
}

// WithContext sets the context for the EPUB parsing operation
func WithContext(ctx context.Context) Option {
	return func(opts *epubOptions) {
		if ctx != nil {
			opts.ctx = ctx
		}
	}
}

// WithCover includes the cover image in the parsed content
func WithCover() Option {
	return func(opts *epubOptions) {
		opts.IncludeCover = true
	}
}

// WithMetadata includes full metadata in the parsed content
func WithMetadata() Option {
	return func(opts *epubOptions) {
		opts.IncludeMetadata = true
	}
}

// WithChapterFilter sets a filter function for chapters
func WithChapterFilter(filter func(chapter Chapter) bool) Option {
	return func(opts *epubOptions) {
		opts.FilterChapters = filter
	}
}

// WithMaxContentLength sets the maximum content length to process
func WithMaxContentLength(maxLen int64) Option {
	return func(opts *epubOptions) {
		opts.MaxContentLength = maxLen
	}
}

// applyOptions applies the given options to the default options
func applyOptions(opts ...Option) *epubOptions {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// isCancelled checks if the context has been cancelled
func (e *epubOptions) isCancelled() bool {
	select {
	case <-e.ctx.Done():
		return true
	default:
		return false
	}
}

// checkContext checks if the context has been cancelled and returns the context error if so
func (e *epubOptions) checkContext() error {
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()
	default:
		return nil
	}
}