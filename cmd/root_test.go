package cmd

import "testing"

func TestNewRootCommandCreatesIndependentGraphs(t *testing.T) {
	first := NewRootCommand()
	second := NewRootCommand()
	if first == second {
		t.Fatal("NewRootCommand returned the same graph")
	}
	if first.Commands()[0] == second.Commands()[0] {
		t.Fatal("command graphs share a command")
	}
	if first.CommandPath() != "gw" || second.CommandPath() != "gw" {
		t.Fatal("unexpected root command")
	}
}
