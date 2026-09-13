package gitcli

import "testing"

func TestNormalizeRemoteURL(t *testing.T) {
	tests := map[string]string{
		"git@github.com:MisterKeke/ForFunPr.git":     "github.com/MisterKeke/ForFunPr",
		"https://github.com/MisterKeke/ForFunPr.git": "github.com/MisterKeke/ForFunPr",
	}
	for input, want := range tests {
		if got := NormalizeRemoteURL(input); got != want {
			t.Fatalf("NormalizeRemoteURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoteWebURL(t *testing.T) {
	got := RemoteWebURL("git@github.com:MisterKeke/ForFunPr.git")
	if got != "https://github.com/MisterKeke/ForFunPr" {
		t.Fatalf("RemoteWebURL returned %q", got)
	}
}
