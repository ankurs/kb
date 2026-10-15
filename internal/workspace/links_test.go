package workspace

import "testing"

func TestReferencesBalancedAndFenced(t *testing.T) {
	md := "# Title\n[x](topic(v2).md) [other](<file name.pdf> \"caption\")\n```md\n[ignored](no.md)\n```\n![image](images/diagram(1).svg)\n"
	refs := References(md)
	if len(refs) != 3 || refs[0].Target != "topic(v2).md" || refs[1].Target != "file name.pdf" || refs[2].Line != 6 || refs[0].Image || refs[1].Image || !refs[2].Image {
		t.Fatalf("refs=%+v", refs)
	}
}

func TestExcalidrawPreviewReferences(t *testing.T) {
	refs := References("![Map](drawings/map.excalidraw) [Source](drawings/map.excalidraw)\n![Drawing](<my drawing.excalidraw> \"title\")\n")
	if len(refs) != 3 || ExcalidrawPreviewKey(refs[0], "drawings/map.excalidraw") != "drawings/map.excalidraw.png" || ExcalidrawPreviewKey(refs[1], "drawings/map.excalidraw") != "" || ExcalidrawPreviewKey(refs[2], "my drawing.excalidraw") != "my drawing.excalidraw.png" {
		t.Fatalf("unexpected preview references: %+v", refs)
	}
}

func TestNestedImageLabels(t *testing.T) {
	md := `![Flow [draft]](flow.excalidraw) [Source [draft]](flow.excalidraw) ![Escaped \[draft\]](flow.excalidraw) \![literal](flow.excalidraw)`
	refs := References(md)
	if len(refs) != 4 || !refs[0].Image || refs[1].Image || !refs[2].Image || refs[3].Image {
		t.Fatalf("nested and escaped labels: %+v", refs)
	}
}
