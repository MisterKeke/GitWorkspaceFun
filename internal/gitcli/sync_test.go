package gitcli

import "testing"

func TestParseSyncCounts(t *testing.T) {
	ahead, behind, err := ParseSyncCounts("2\t3\n")
	if err != nil {
		t.Fatalf("ParseSyncCounts returned an error: %v", err)
	}
	if ahead != 2 || behind != 3 {
		t.Fatalf("ParseSyncCounts = %d/%d, want 2/3", ahead, behind)
	}
}
