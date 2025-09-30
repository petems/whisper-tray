package inject

import "context"

// Injector defines the interface for text injection
type Injector interface {
	Paste(ctx context.Context, text string) error
	Type(ctx context.Context, text string) error
	PasteOrType(ctx context.Context, text string) error
}