package gitcli

import (
	"reflect"
	"testing"
)

func TestParseChanges(t *testing.T) {
	changes, err := ParseChanges(" M modified.go\nA  added.go\n D deleted.go\nR  old.go -> new.go\n?? untracked.go\n")
	if err != nil {
		t.Fatalf("ParseChanges returned an error: %v", err)
	}

	want := Changes{
		Modified:  1,
		Added:     1,
		Deleted:   1,
		Renamed:   1,
		Untracked: 1,
		Lines: []string{
			" M modified.go",
			"A  added.go",
			" D deleted.go",
			"R  old.go -> new.go",
			"?? untracked.go",
		},
	}
	if !reflect.DeepEqual(changes, want) {
		t.Fatalf("ParseChanges returned %+v, want %+v", changes, want)
	}
}

func TestParseChangesClean(t *testing.T) {
	changes, err := ParseChanges("")
	if err != nil {
		t.Fatalf("ParseChanges returned an error: %v", err)
	}
	if changes.IsDirty() {
		t.Fatalf("empty status should be clean: %+v", changes)
	}
}
