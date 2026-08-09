package api

import (
	"strings"
	"testing"
)

// BuiltinSearchEngines feeds both first-start seeding and restore-builtins, so
// a malformed entry would ship broken engines to every new deployment. These
// are pure-function assertions — no DB involved.
func TestBuiltinSearchEngines(t *testing.T) {
	engines := BuiltinSearchEngines()

	if len(engines) != 17 {
		t.Fatalf("expected 17 builtin engines, got %d", len(engines))
	}

	defaults := 0
	seen := map[string]bool{}
	byCategory := map[string]int{}
	for _, e := range engines {
		if e.IsDefault {
			defaults++
			if e.Name != "Google" {
				t.Errorf("default engine should be Google, got %q", e.Name)
			}
			if e.Category != CategoryWeb {
				t.Errorf("default engine should be in category web, got %q", e.Category)
			}
		}
		if strings.TrimSpace(e.Name) == "" {
			t.Errorf("engine with empty name: %+v", e)
			continue
		}
		if seen[e.Name] {
			t.Errorf("duplicate engine name: %q", e.Name)
		}
		seen[e.Name] = true

		if !strings.HasPrefix(e.URLTemplate, "https://") {
			t.Errorf("%s: url_template must start with https://, got %q", e.Name, e.URLTemplate)
		}
		if !placeholderPattern.MatchString(e.URLTemplate) {
			t.Errorf("%s: url_template missing {q}/{query} placeholder: %q", e.Name, e.URLTemplate)
		}

		if e.Category == "" {
			t.Errorf("%s: category must not be empty", e.Name)
		} else if !IsValidSearchEngineCategory(e.Category) {
			t.Errorf("%s: unknown category %q", e.Name, e.Category)
		}
		byCategory[e.Category]++
	}

	if defaults != 1 {
		t.Errorf("expected exactly 1 default engine, got %d", defaults)
	}

	want := map[string]int{
		CategoryWeb:   7,
		CategoryImage: 4,
		CategoryMusic: 3,
		CategoryVideo: 3,
	}
	for cat, n := range want {
		if byCategory[cat] != n {
			t.Errorf("category %s: expected %d engines, got %d", cat, n, byCategory[cat])
		}
	}
}

func TestSearchEngineCategories(t *testing.T) {
	got := SearchEngineCategories()
	want := []string{"web", "image", "music", "video"}
	if len(got) != len(want) {
		t.Fatalf("expected %d categories, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("category order[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestIsValidSearchEngineCategory(t *testing.T) {
	for _, c := range SearchEngineCategories() {
		if !IsValidSearchEngineCategory(c) {
			t.Errorf("%q should be valid", c)
		}
	}
	// Empty is deliberately invalid — create/update decide whether to default
	// it, the validator itself must not silently accept a blank.
	for _, c := range []string{"", "Web", "WEB", "news", "web ", "image;drop"} {
		if IsValidSearchEngineCategory(c) {
			t.Errorf("%q should be invalid", c)
		}
	}
}

func TestValidateSearchEngineWriteCategory(t *testing.T) {
	base := func() *searchEngineWriteRequest {
		return &searchEngineWriteRequest{Name: "X", URLTemplate: "https://e.com/?q={query}"}
	}

	t.Run("omitted category is accepted", func(t *testing.T) {
		if msg := validateSearchEngineWrite(base(), true); msg != "" {
			t.Errorf("expected no error, got %q", msg)
		}
	})

	for _, c := range SearchEngineCategories() {
		t.Run("valid "+c, func(t *testing.T) {
			req := base()
			req.Category = c
			if msg := validateSearchEngineWrite(req, true); msg != "" {
				t.Errorf("expected no error for %q, got %q", c, msg)
			}
		})
	}

	t.Run("unknown category is rejected", func(t *testing.T) {
		req := base()
		req.Category = "news"
		if msg := validateSearchEngineWrite(req, true); msg == "" {
			t.Error("expected an error for category=news")
		}
	})
}
