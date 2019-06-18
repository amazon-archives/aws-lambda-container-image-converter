package prioritize

import (
	"fmt"
	"testing"
	"time"

	"github.com/containers/image/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

const (
	digestUnknown             = digest.Digest("sha256:1111111111111111111111111111111111111111111111111111111111111111")
	digestUncompressed        = digest.Digest("sha256:2222222222222222222222222222222222222222222222222222222222222222")
	digestCompressedA         = digest.Digest("sha256:3333333333333333333333333333333333333333333333333333333333333333")
	digestCompressedB         = digest.Digest("sha256:4444444444444444444444444444444444444444444444444444444444444444")
	digestCompressedUnrelated = digest.Digest("sha256:5555555555555555555555555555555555555555555555555555555555555555")
	digestCompressedPrimary   = digest.Digest("sha256:6666666666666666666666666666666666666666666666666666666666666666")
)

var (
	// cssLiteral contains a non-trivial candidateSortState shared among several tests below.
	cssLiteral = candidateSortState{
		cs: []CandidateWithTime{
			{types.BICReplacementCandidate{Digest: digestCompressedA, Location: types.BICLocationReference{Opaque: "A1"}}, time.Unix(1, 0)},
			{types.BICReplacementCandidate{Digest: digestUncompressed, Location: types.BICLocationReference{Opaque: "U2"}}, time.Unix(1, 1)},
			{types.BICReplacementCandidate{Digest: digestCompressedA, Location: types.BICLocationReference{Opaque: "A2"}}, time.Unix(1, 1)},
			{types.BICReplacementCandidate{Digest: digestCompressedPrimary, Location: types.BICLocationReference{Opaque: "P1"}}, time.Unix(1, 0)},
			{types.BICReplacementCandidate{Digest: digestCompressedB, Location: types.BICLocationReference{Opaque: "B1"}}, time.Unix(1, 1)},
			{types.BICReplacementCandidate{Digest: digestCompressedPrimary, Location: types.BICLocationReference{Opaque: "P2"}}, time.Unix(1, 1)},
			{types.BICReplacementCandidate{Digest: digestCompressedB, Location: types.BICLocationReference{Opaque: "B2"}}, time.Unix(2, 0)},
			{types.BICReplacementCandidate{Digest: digestUncompressed, Location: types.BICLocationReference{Opaque: "U1"}}, time.Unix(1, 0)},
		},
		primaryDigest:      digestCompressedPrimary,
		uncompressedDigest: digestUncompressed,
	}
	// cssExpectedReplacementCandidates is the fully-sorted, unlimited, result of prioritizing cssLiteral.
	cssExpectedReplacementCandidates = []types.BICReplacementCandidate{
		{Digest: digestCompressedPrimary, Location: types.BICLocationReference{Opaque: "P2"}},
		{Digest: digestCompressedPrimary, Location: types.BICLocationReference{Opaque: "P1"}},
		{Digest: digestCompressedB, Location: types.BICLocationReference{Opaque: "B2"}},
		{Digest: digestCompressedA, Location: types.BICLocationReference{Opaque: "A2"}},
		{Digest: digestCompressedB, Location: types.BICLocationReference{Opaque: "B1"}},
		{Digest: digestCompressedA, Location: types.BICLocationReference{Opaque: "A1"}},
		{Digest: digestUncompressed, Location: types.BICLocationReference{Opaque: "U2"}},
		{Digest: digestUncompressed, Location: types.BICLocationReference{Opaque: "U1"}},
	}
)

func TestCandidateSortStateLen(t *testing.T) {
	css := cssLiteral
	assert.Equal(t, 8, css.Len())

	css.cs = []CandidateWithTime{}
	assert.Equal(t, 0, css.Len())
}

func TestCandidateSortStateLess(t *testing.T) {
	type p struct {
		d digest.Digest
		t int64
	}

	// Primary criteria: Also ensure that time does not matter
	for _, c := range []struct {
		name   string
		res    int
		d0, d1 digest.Digest
	}{
		{"primary < any", -1, digestCompressedPrimary, digestCompressedA},
		{"any < uncompressed", -1, digestCompressedA, digestUncompressed},
		{"primary < uncompressed", -1, digestCompressedPrimary, digestUncompressed},
	} {
		for _, tms := range [][2]int64{{1, 2}, {2, 1}, {1, 1}} {
			caseName := fmt.Sprintf("%s %v", c.name, tms)
			css := candidateSortState{
				cs: []CandidateWithTime{
					{types.BICReplacementCandidate{Digest: c.d0, Location: types.BICLocationReference{Opaque: "L0"}}, time.Unix(tms[0], 0)},
					{types.BICReplacementCandidate{Digest: c.d1, Location: types.BICLocationReference{Opaque: "L1"}}, time.Unix(tms[1], 0)},
				},
				primaryDigest:      digestCompressedPrimary,
				uncompressedDigest: digestUncompressed,
			}
			assert.Equal(t, c.res < 0, css.Less(0, 1), caseName)
			assert.Equal(t, c.res > 0, css.Less(1, 0), caseName)

			if c.d0 != digestUncompressed && c.d1 != digestUncompressed {
				css.uncompressedDigest = ""
				assert.Equal(t, c.res < 0, css.Less(0, 1), caseName)
				assert.Equal(t, c.res > 0, css.Less(1, 0), caseName)

				css.uncompressedDigest = css.primaryDigest
				assert.Equal(t, c.res < 0, css.Less(0, 1), caseName)
				assert.Equal(t, c.res > 0, css.Less(1, 0), caseName)
			}
		}
	}

	// Ordering within the three primary groups
	for _, c := range []struct {
		name   string
		res    int
		p0, p1 p
	}{
		{"primary: t=2 < t=1", -1, p{digestCompressedPrimary, 2}, p{digestCompressedPrimary, 1}},
		{"primary: t=1 == t=1", 0, p{digestCompressedPrimary, 1}, p{digestCompressedPrimary, 1}},
		{"uncompressed: t=2 < t=1", -1, p{digestUncompressed, 2}, p{digestUncompressed, 1}},
		{"uncompressed: t=1 == t=1", 0, p{digestUncompressed, 1}, p{digestUncompressed, 1}},
		{"any: t=2 < t=1, [d=A vs. d=B lower-priority]", -1, p{digestCompressedA, 2}, p{digestCompressedB, 1}},
		{"any: t=2 < t=1, [d=B vs. d=A lower-priority]", -1, p{digestCompressedB, 2}, p{digestCompressedA, 1}},
		{"any: t=2 < t=1, [d=A vs. d=A lower-priority]", -1, p{digestCompressedA, 2}, p{digestCompressedA, 1}},
		{"any: t=1 == t=1, d=A < d=B", -1, p{digestCompressedA, 1}, p{digestCompressedB, 1}},
		{"any: t=1 == t=1, d=A == d=A", 0, p{digestCompressedA, 1}, p{digestCompressedA, 1}},
	} {
		css := candidateSortState{
			cs: []CandidateWithTime{
				{types.BICReplacementCandidate{Digest: c.p0.d, Location: types.BICLocationReference{Opaque: "L0"}}, time.Unix(c.p0.t, 0)},
				{types.BICReplacementCandidate{Digest: c.p1.d, Location: types.BICLocationReference{Opaque: "L1"}}, time.Unix(c.p1.t, 0)},
			},
			primaryDigest:      digestCompressedPrimary,
			uncompressedDigest: digestUncompressed,
		}
		assert.Equal(t, c.res < 0, css.Less(0, 1), c.name)
		assert.Equal(t, c.res > 0, css.Less(1, 0), c.name)

		if c.p0.d != digestUncompressed && c.p1.d != digestUncompressed {
			css.uncompressedDigest = ""
			assert.Equal(t, c.res < 0, css.Less(0, 1), c.name)
			assert.Equal(t, c.res > 0, css.Less(1, 0), c.name)

			css.uncompressedDigest = css.primaryDigest
			assert.Equal(t, c.res < 0, css.Less(0, 1), c.name)
			assert.Equal(t, c.res > 0, css.Less(1, 0), c.name)
		}
	}
}

func TestCandidateSortStateSwap(t *testing.T) {
	freshCSS := func() candidateSortState { // Return a deep copy of cssLiteral which is safe to modify.
		res := cssLiteral
		res.cs = append([]CandidateWithTime{}, cssLiteral.cs...)
		return res
	}

	css := freshCSS()
	css.Swap(0, 1)
	assert.Equal(t, cssLiteral.cs[1], css.cs[0])
	assert.Equal(t, cssLiteral.cs[0], css.cs[1])
	assert.Equal(t, cssLiteral.cs[2], css.cs[2])

	css = freshCSS()
	css.Swap(1, 1)
	assert.Equal(t, cssLiteral, css)
}

func TestDestructivelyPrioritizeReplacementCandidatesWithMax(t *testing.T) {
	for _, max := range []int{0, 1, replacementAttempts, 100} {
		// Just a smoke test; we mostly rely on test coverage in TestCandidateSortStateLess
		res := destructivelyPrioritizeReplacementCandidatesWithMax(append([]CandidateWithTime{}, cssLiteral.cs...), digestCompressedPrimary, digestUncompressed, max)
		if max > len(cssExpectedReplacementCandidates) {
			max = len(cssExpectedReplacementCandidates)
		}
		assert.Equal(t, cssExpectedReplacementCandidates[:max], res)
	}
}

func TestDestructivelyPrioritizeReplacementCandidates(t *testing.T) {
	// Just a smoke test; we mostly rely on test coverage in TestCandidateSortStateLess
	res := DestructivelyPrioritizeReplacementCandidates(append([]CandidateWithTime{}, cssLiteral.cs...), digestCompressedPrimary, digestUncompressed)
	assert.Equal(t, cssExpectedReplacementCandidates[:replacementAttempts], res)
}
