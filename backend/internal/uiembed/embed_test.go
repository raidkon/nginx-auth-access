package uiembed

import (
	"io/fs"
	"testing"
)

func TestSPARootHasIndexHTML(t *testing.T) {
	root, err := SPARoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(root, "index.html"); err != nil {
		t.Fatalf("index.html missing: %v", err)
	}
}
