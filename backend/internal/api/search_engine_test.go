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

	if len(engines) != 7 {
		t.Fatalf("expected 7 builtin engines, got %d", len(engines))
	}

	defaults := 0
	seen := map[string]bool{}
	for _, e := range engines {
		if e.IsDefault {
			defaults++
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
	}

	if defaults != 1 {
		t.Errorf("expected exactly 1 default engine, got %d", defaults)
	}
}
