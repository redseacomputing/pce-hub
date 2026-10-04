// ins Pakus internal/controller als sanity_test.go kopieren
package controller

import (
	"testing"

	"github.com/redseacomputing/pce/pkg/mapping"
)

func TestCrosswalkLibraryAvailable(t *testing.T) {
	engine, err := mapping.NewDefaultCrosswalkEngine()
	if err != nil {
		t.Fatalf("library bootstrap failed: %v", err)
	}
	controls := engine.GetControlsForArticle(mapping.Art21_2h_CryptoEncryption)
	if len(controls) != 4 {
		t.Fatalf("expected 4 controls for lit. h, got %d", len(controls))
	}
}
