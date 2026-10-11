package app_test

import (
	"os"
	"strings"
	"testing"
)

// Variant describes one student's resource. Every variant has the same shape:
// four required string fields, one optional integer and one optional text field.
// The filter field is always the fourth required field.
type Variant struct {
	Name     string
	Resource string
	Required []string
	OptInt   string
	OptText  string
}

// Filter returns the name of the field used both as a JSON field and as the list filter.
func (v Variant) Filter() string {
	return v.Required[3]
}

var variants = []Variant{
	{Name: "books", Resource: "books", Required: []string{"title", "isbn", "author", "category"}, OptInt: "published_year", OptText: "description"},
	{Name: "movies", Resource: "movies", Required: []string{"title", "director", "country", "genre"}, OptInt: "release_year", OptText: "synopsis"},
	{Name: "tasks", Resource: "tasks", Required: []string{"title", "assignee", "project", "status"}, OptInt: "priority", OptText: "notes"},
	{Name: "recipes", Resource: "recipes", Required: []string{"name", "author", "difficulty", "cuisine"}, OptInt: "cook_minutes", OptText: "instructions"},
	{Name: "devices", Resource: "devices", Required: []string{"name", "serial", "manufacturer", "type"}, OptInt: "warranty_months", OptText: "comment"},
	{Name: "courses", Resource: "courses", Required: []string{"title", "teacher", "level", "subject"}, OptInt: "hours", OptText: "program"},
	{Name: "albums", Resource: "albums", Required: []string{"title", "artist", "label", "genre"}, OptInt: "release_year", OptText: "notes"},
	{Name: "pets", Resource: "pets", Required: []string{"name", "owner", "breed", "species"}, OptInt: "age", OptText: "notes"},
}

// mustVariant returns the variant chosen in the VARIANT file at the repository root.
// The environment variable VARIANT, if set, takes precedence (used by graders).
func mustVariant(t *testing.T) Variant {
	t.Helper()

	name := strings.ToLower(strings.TrimSpace(os.Getenv("VARIANT")))
	if name == "" {
		data, err := os.ReadFile("../../VARIANT")
		if err != nil {
			t.Fatalf("cannot read the VARIANT file in the repository root: %v", err)
		}
		name = strings.ToLower(strings.TrimSpace(string(data)))
	}

	names := make([]string, 0, len(variants))
	for _, v := range variants {
		if v.Name == name {
			return v
		}
		names = append(names, v.Name)
	}

	t.Fatalf("unknown variant %q: put one of these names into the VARIANT file: %s", name, strings.Join(names, ", "))
	return Variant{}
}
