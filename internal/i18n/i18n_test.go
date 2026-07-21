package i18n

import (
	"sort"
	"testing"
)

func TestParseDefaultZH(t *testing.T) {
	if Parse("") != ZH {
		t.Fatal("empty should be zh")
	}
	if Parse("en") != EN {
		t.Fatal("en")
	}
	if Parse("EN-US") != EN {
		t.Fatal("EN-US")
	}
	if Parse("zh-CN") != ZH {
		t.Fatal("zh-CN")
	}
}

func TestCatalogFallback(t *testing.T) {
	zh := New(ZH)
	en := New(EN)
	if zh.S("scan.summary_title") == en.S("scan.summary_title") {
		t.Fatal("expected different languages")
	}
	if zh.ToolLabel("agents") != "真源" {
		t.Fatalf("got %s", zh.ToolLabel("agents"))
	}
	if en.ToolLabel("agents") != "agents" {
		t.Fatalf("got %s", en.ToolLabel("agents"))
	}
}

func TestZHENKeyParity(t *testing.T) {
	// Keep UI catalogs in sync so --en / --zh never miss a string.
	onlyZH, onlyEN := []string{}, []string{}
	for k := range zh {
		if _, ok := en[k]; !ok {
			onlyZH = append(onlyZH, k)
		}
	}
	for k := range en {
		if _, ok := zh[k]; !ok {
			onlyEN = append(onlyEN, k)
		}
	}
	sort.Strings(onlyZH)
	sort.Strings(onlyEN)
	if len(onlyZH) > 0 || len(onlyEN) > 0 {
		t.Fatalf("i18n key mismatch\nonly zh: %v\nonly en: %v", onlyZH, onlyEN)
	}
	// Recent bilingual features must differ between locales.
	for _, key := range []string{
		"gate.msg.global_off",
		"gate.msg.tools_off",
		"plan.note.tool_off",
		"doctor.prov_none",
		"outdated.msg.no_prov",
		"outdated.msg.lock_ok",
		"main.usage",
	} {
		if zh[key] == "" || en[key] == "" {
			t.Fatalf("missing %s", key)
		}
		if zh[key] == en[key] {
			t.Fatalf("%s should be translated (zh==en)", key)
		}
	}
}
