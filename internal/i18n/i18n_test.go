package i18n

import "testing"

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
