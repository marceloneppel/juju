package charm

import (
	"testing"
)

func TestSelectDefaultEndpointPair(t *testing.T) {
	cases := []struct {
		name    string
		pairs   [][2]bool
		wantIdx int
		wantOK  bool
	}{
		{"none", [][2]bool{{false, false}, {false, false}}, 0, false},
		{"one-left", [][2]bool{{true, false}, {false, false}}, 0, true},
		{"one-right", [][2]bool{{false, false}, {false, true}}, 1, true},
		{"both-sides-one-pair", [][2]bool{{true, true}, {false, false}}, 0, true},
		{"multiple", [][2]bool{{true, false}, {false, true}}, 0, false},
		{"empty", [][2]bool{}, 0, false},
	}
	for _, tc := range cases {
		idx, ok := SelectDefaultEndpointPair(tc.pairs)
		if ok != tc.wantOK || (ok && idx != tc.wantIdx) {
			t.Errorf("%s: got (%d,%v), want (%d,%v)", tc.name, idx, ok, tc.wantIdx, tc.wantOK)
		}
	}
}
