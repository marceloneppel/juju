// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

// SelectDefaultEndpointPair disambiguates a set of candidate endpoint pairs by
// the per-endpoint default flag. Each element is {ep1IsDefault, ep2IsDefault}.
// It returns the index of the sole pair that has a default-marked endpoint, and
// true, iff exactly one such pair exists. Zero or multiple default pairs return
// ok=false, leaving the caller's existing ambiguity handling in place.
//
// It is shared by the live relation inference (domain/relation/state) and the
// bundle inference (this package) so the single-winner rule is defined once.
func SelectDefaultEndpointPair(pairs [][2]bool) (idx int, ok bool) {
	selected, count := 0, 0
	for i, p := range pairs {
		if p[0] || p[1] {
			selected, count = i, count+1
		}
	}
	return selected, count == 1
}
