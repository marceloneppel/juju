// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"testing"

	"github.com/juju/tc"

	"github.com/juju/juju/domain/deployment/charm"
)

type endpointDefaultSuite struct{}

func TestEndpointDefaultSuite(t *testing.T) {
	tc.Run(t, &endpointDefaultSuite{})
}

func (s *endpointDefaultSuite) TestSelectDefaultEndpointPair(c *tc.C) {
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
	for _, test := range cases {
		idx, ok := charm.SelectDefaultEndpointPair(test.pairs)
		c.Check(ok, tc.Equals, test.wantOK, tc.Commentf("%s", test.name))
		if test.wantOK {
			c.Check(idx, tc.Equals, test.wantIdx, tc.Commentf("%s", test.name))
		}
	}
}
