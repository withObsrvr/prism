package componentsv2

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDetailHeaderUsesSharedPrimaryNavigation(t *testing.T) {
	var out bytes.Buffer
	if err := DetailHeader("contracts", "CABC", "testnet").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, required := range []string{"Prism home", ">Home</a>", ">Explore</a>", ">Assets</a>", ">Contracts</a>", ">Docs</a>", `aria-label="Mobile navigation"`, `value="CABC"`} {
		if !strings.Contains(html, required) {
			t.Errorf("detail header missing %q", required)
		}
	}
	if !strings.Contains(html, `class="active" href="/contracts"`) {
		t.Error("detail header does not mark the current primary section")
	}
}
