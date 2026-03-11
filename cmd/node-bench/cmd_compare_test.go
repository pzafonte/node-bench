package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pzafonte/node-bench/internal/result"
)

// TestMergedCheckpointHeights covers the union + numeric-sort logic.
func TestMergedCheckpointHeights(t *testing.T) {
	v := 42
	tests := []struct {
		name string
		a, b result.Checkpoints
		want []int
	}{
		{
			name: "both empty",
			a:    result.Checkpoints{},
			b:    result.Checkpoints{},
			want: []int{},
		},
		{
			name: "only a has keys",
			a:    result.Checkpoints{"1000": &v, "5000": nil},
			b:    result.Checkpoints{},
			want: []int{1000, 5000},
		},
		{
			name: "only b has keys",
			a:    result.Checkpoints{},
			b:    result.Checkpoints{"500": &v, "2000": nil},
			want: []int{500, 2000},
		},
		{
			name: "union deduplicates shared key",
			a:    result.Checkpoints{"1000": &v},
			b:    result.Checkpoints{"1000": nil, "5000": nil},
			want: []int{1000, 5000},
		},
		{
			name: "sorts numerically not lexicographically",
			// Lexicographic sort would put "100" before "50"; numeric must give [50,100].
			a:    result.Checkpoints{"100": &v},
			b:    result.Checkpoints{"50": nil},
			want: []int{50, 100},
		},
		{
			name: "custom list differs from defaults",
			a:    result.Checkpoints{"250": &v, "750": nil},
			b:    result.Checkpoints{"250": nil, "1500": &v},
			want: []int{250, 750, 1500},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergedCheckpointHeights(tc.a, tc.b)
			if len(got) != len(tc.want) {
				t.Fatalf("mergedCheckpointHeights len = %d, want %d: got %v", len(got), len(tc.want), got)
			}
			for i, h := range got {
				if h != tc.want[i] {
					t.Errorf("got[%d] = %d, want %d", i, h, tc.want[i])
				}
			}
		})
	}
}

// TestPrintComparison verifies the compare output contains the expected
// structural elements and uses only the checkpoint keys present in the results.
func TestPrintComparison(t *testing.T) {
	cp250a := 30
	cp750a := 90
	cp250b := 25

	rA := &result.Result{
		Commit:       "aaa1111",
		Branch:       "master",
		HeaderSyncS:  200,
		BlockSyncS:   100,
		BlocksPerSec: 3.0,
		MaxHeight:    10000,
		Checkpoints:  result.Checkpoints{"250": &cp250a, "750": &cp750a},
	}
	rB := &result.Result{
		Commit:       "bbb2222",
		Branch:       "feature",
		HeaderSyncS:  180,
		BlockSyncS:   110,
		BlocksPerSec: 3.5,
		MaxHeight:    10500,
		Checkpoints:  result.Checkpoints{"250": &cp250b},
	}

	var buf bytes.Buffer
	printComparison(&buf, "aaa1111", "bbb2222", rA, rB)
	out := buf.String()

	if !strings.Contains(out, "aaa1111") {
		t.Errorf("output missing sha-a: %s", out)
	}
	if !strings.Contains(out, "bbb2222") {
		t.Errorf("output missing sha-b: %s", out)
	}

	// Checkpoint rows must come from the union of both result key sets.
	// Height 250 is in both; height 750 is only in rA.
	if !strings.Contains(out, "250") {
		t.Errorf("output missing checkpoint height 250: %s", out)
	}
	if !strings.Contains(out, "750") {
		t.Errorf("output missing checkpoint height 750 (from rA only): %s", out)
	}

	// Height 750 is not reached in rB — must say "not reached".
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "750") {
			if !strings.Contains(line, "not reached") {
				t.Errorf("line for height 750 should show 'not reached' for rB: %q", line)
			}
		}
	}

	// Δ for header sync: 200-->180 is a 10% improvement (lower is better --> +10%).
	if !strings.Contains(out, "+10.0%") {
		t.Errorf("expected +10.0%% delta for header sync, output:\n%s", out)
	}
}
