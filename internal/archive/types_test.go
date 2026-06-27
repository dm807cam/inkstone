package archive

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOrderedPagesPrefersCPages(t *testing.T) {
	// A modern (formatVersion 2) .content where the device order differs from
	// the order .rm files would sort alphabetically by UUID.
	raw := `{
		"formatVersion": 2,
		"cPages": {
			"pages": [
				{"id": "page-c", "idx": {"value": "ba"}},
				{"id": "page-a", "idx": {"value": "bb"}},
				{"id": "page-b", "idx": {"value": "bc"}}
			]
		}
	}`

	var c Content
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := c.OrderedPages()
	want := []string{"page-c", "page-a", "page-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OrderedPages() = %v, want %v", got, want)
	}
}

func TestOrderedPagesSkipsDeleted(t *testing.T) {
	raw := `{
		"cPages": {
			"pages": [
				{"id": "p1"},
				{"id": "p2", "deleted": {"value": 1}},
				{"id": "p3"}
			]
		}
	}`

	var c Content
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := c.OrderedPages()
	want := []string{"p1", "p3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OrderedPages() = %v, want %v", got, want)
	}
}

func TestOrderedPagesFallsBackToLegacyPages(t *testing.T) {
	raw := `{"pages": ["a", "b", "c"]}`

	var c Content
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := c.OrderedPages()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OrderedPages() = %v, want %v", got, want)
	}
}
