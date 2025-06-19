package main

import "testing"

func TestContainsWord(t *testing.T) {
	tests := []struct {
		text string
		word string
		want bool
	}{
		{"Add one mana of any color", "mana", true},
		{"Add One Mana of any color", "mana", true},
		{"The manacost is high", "mana", false},
		{"Check the mana-cost here", "mana", true},
	}

	for _, tt := range tests {
		if got := containsWord(tt.text, tt.word); got != tt.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, got, tt.want)
		}
	}
}

func TestContainsAnyPhrase(t *testing.T) {
	tests := []struct {
		text    string
		phrases []string
		want    bool
	}{
		{"draw a card and you may draw two", []string{"counter target", "draw a card"}, true},
		{"Destroy target creature", []string{"destroy target creature"}, false},
		{"Destroy target creature", []string{"Destroy target creature"}, true},
		{"Destroy target creature", []string{"target creature", "random"}, true},
		{"Destroy target creature", []string{"destroy Target creature"}, false},
	}

	for _, tt := range tests {
		if got := containsAnyPhrase(tt.text, tt.phrases...); got != tt.want {
			t.Errorf("containsAnyPhrase(%q, %q) = %v, want %v", tt.text, tt.phrases, got, tt.want)
		}
	}
}

func TestCountManaPips(t *testing.T) {
	tests := []struct {
		manaCost string
		total    int
		expect   map[string]int
	}{
		{"{3}{R}{G}", 3, map[string]int{"3": 1, "R": 1, "G": 1}},
		{"{W}{U/B}{U/B}", 5, map[string]int{"W": 1, "U": 2, "B": 2}},
		{"{2}{G/U}{G/U}", 5, map[string]int{"2": 1, "G": 2, "U": 2}},
		{"{X}{B}", 2, map[string]int{"X": 1, "B": 1}},
	}

	for _, tt := range tests {
		total, counts := countManaPips(tt.manaCost)
		if total != tt.total {
			t.Errorf("countManaPips(%q) total=%d, want %d", tt.manaCost, total, tt.total)
		}
		for sym, wantCount := range tt.expect {
			if got := counts[sym]; got != wantCount {
				t.Errorf("countManaPips(%q) symbol %s=%d, want %d", tt.manaCost, sym, got, wantCount)
			}
		}
	}
}
