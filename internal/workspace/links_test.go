package workspace

import "testing"

func TestReferencesBalancedAndFenced(t *testing.T) {
	md := "# Title\n[x](topic(v2).md) [other](<file name.pdf> \"caption\")\n```md\n[ignored](no.md)\n```\n![image](images/diagram(1).svg)\n"
	refs := References(md)
	if len(refs) != 3 || refs[0].Target != "topic(v2).md" || refs[1].Target != "file name.pdf" || refs[2].Line != 6 {
		t.Fatalf("refs=%+v", refs)
	}
}
