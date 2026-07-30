package library_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func TestWriteFileNewDoesNotReplaceExistingContent(t *testing.T) {
	contentLibrary, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err = contentLibrary.WriteFileNew(
		library.KindWalks,
		"captured/switch.walk",
		[]byte("first"),
	); err != nil {
		t.Fatalf("first WriteFileNew() error = %v", err)
	}
	if err = contentLibrary.WriteFileNew(
		library.KindWalks,
		"captured/switch.walk",
		[]byte("second"),
	); !errors.Is(err, library.ErrAlreadyExists) {
		t.Fatalf("duplicate WriteFileNew() error = %v, want ErrAlreadyExists", err)
	}
	content, err := contentLibrary.ReadFile(library.KindWalks, "captured/switch.walk")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "first" {
		t.Fatalf("content = %q, want first", content)
	}
}
