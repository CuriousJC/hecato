package files

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

func sizes(files []File) []int64 {
	out := make([]int64, len(files))
	for i, f := range files {
		out[i] = f.Size
	}
	return out
}

func equalInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTopNKeepsTheLargest(t *testing.T) {
	top := newTopN(3, worseBySize)
	for _, n := range []int64{5, 1, 9, 3, 7, 2} {
		top.add(File{Size: n})
	}

	if got, want := sizes(top.sorted()), []int64{9, 7, 5}; !equalInt64(got, want) {
		t.Errorf("sorted() = %v, want %v", got, want)
	}
}

func TestTopNKeepsTheMostRecent(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	top := newTopN(2, worseByMod)

	for _, d := range []int{5, 1, 9, 3} {
		top.add(File{ModTime: base.AddDate(0, 0, d)})
	}

	got := top.sorted()
	if len(got) != 2 {
		t.Fatalf("got %d files, want 2", len(got))
	}
	if !got[0].ModTime.Equal(base.AddDate(0, 0, 9)) {
		t.Errorf("first = %v, want day 9", got[0].ModTime)
	}
	if !got[1].ModTime.Equal(base.AddDate(0, 0, 5)) {
		t.Errorf("second = %v, want day 5", got[1].ModTime)
	}
}

func TestTopNFewerItemsThanLimit(t *testing.T) {
	top := newTopN(10, worseBySize)
	top.add(File{Size: 2})
	top.add(File{Size: 1})

	if got, want := sizes(top.sorted()), []int64{2, 1}; !equalInt64(got, want) {
		t.Errorf("sorted() = %v, want %v", got, want)
	}
}

func TestTopNZeroAndNegativeLimits(t *testing.T) {
	for _, limit := range []int{0, -1, -100} {
		top := newTopN(limit, worseBySize)
		for i := 0; i < 5; i++ {
			top.add(File{Size: int64(i)})
		}
		if got := top.sorted(); len(got) != 0 {
			t.Errorf("limit %d kept %d files, want 0", limit, len(got))
		}
	}
}

// The scan gives each worker its own topN and merges at the end. Merging must
// give the same answer as if one collector had seen everything, or the result
// would depend on which worker happened to see which file.
func TestTopNMergeMatchesSingleCollector(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	all := make([]int64, 500)
	for i := range all {
		all[i] = r.Int63n(10000)
	}

	const limit = 12

	single := newTopN(limit, worseBySize)
	for _, n := range all {
		single.add(File{Size: n})
	}

	// Same values, dealt across four collectors, then merged.
	shards := make([]*topN, 4)
	for i := range shards {
		shards[i] = newTopN(limit, worseBySize)
	}
	for i, n := range all {
		shards[i%4].add(File{Size: n})
	}

	merged := newTopN(limit, worseBySize)
	for _, s := range shards {
		merged.merge(s)
	}

	want := sizes(single.sorted())
	got := sizes(merged.sorted())

	if !equalInt64(got, want) {
		t.Errorf("merged  = %v\nsingle  = %v", got, want)
	}

	// And both must agree with a plain sort of everything.
	sort.Slice(all, func(i, j int) bool { return all[i] > all[j] })
	if !equalInt64(want, all[:limit]) {
		t.Errorf("single collector = %v, want %v", want, all[:limit])
	}
}

func TestTopNMergeNilIsSafe(t *testing.T) {
	top := newTopN(3, worseBySize)
	top.add(File{Size: 1})
	top.merge(nil)

	if got := top.sorted(); len(got) != 1 {
		t.Errorf("merging nil changed the contents: %v", sizes(got))
	}
}

// A limit far larger than the number of files must not preallocate for the
// limit. This is a behaviour check rather than a memory one: it simply has to
// work without falling over.
func TestTopNHugeLimit(t *testing.T) {
	top := newTopN(10_000_000, worseBySize)
	top.add(File{Size: 42})

	if got := top.sorted(); len(got) != 1 || got[0].Size != 42 {
		t.Errorf("sorted() = %v, want one file of size 42", sizes(got))
	}
}
