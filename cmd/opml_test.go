package cmd

import (
	"strings"
	"testing"
)

func TestExportCmd(t *testing.T) {
	tsURL := setupTest(t)

	_, _ = executeCommand("add", tsURL+"/feed.xml")

	out, err := executeCommand("export")
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
	if !strings.Contains(out, "<opml") {
		t.Errorf("expected OPML output: %s", out)
	}
	if !strings.Contains(out, "xmlUrl=") {
		t.Errorf("expected feed URL in OPML: %s", out)
	}
}

func TestImportCmd(t *testing.T) {
	tsURL := setupTest(t)

	opml := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Feed" type="rss" xmlUrl="` + tsURL + `/feed.xml"/>
  </body>
</opml>`

	file := writeTempFile(t, "import-*.opml", opml)
	out, err := executeCommand("import", file)
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}
	if !strings.Contains(out, "Imported 1 feeds") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestImportCmdJSON(t *testing.T) {
	tsURL := setupTest(t)

	opml := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Feed" type="rss" xmlUrl="` + tsURL + `/feed.xml"/>
  </body>
</opml>`

	file := writeTempFile(t, "import-*.opml", opml)
	out, err := executeCommand("import", file, "--json")
	if err != nil {
		t.Fatalf("import --json failed: %v", err)
	}
	if !strings.Contains(out, `"added_count"`) {
		t.Errorf("expected JSON output: %s", out)
	}
}
