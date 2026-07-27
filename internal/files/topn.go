package files

import (
	"container/heap"
	"sort"
)

// worseFunc reports whether a should be discarded before b under the ordering
// the caller cares about. For largefiles that means smaller; for modfiles,
// older. It is the inverse of "interesting", which is why it reads backwards.
type worseFunc func(a, b File) bool

// fileHeap is a min-heap on worseFunc, so the element to evict is always at the
// root. Only topN uses it; the heap.Interface methods are not meant to be
// called directly.
type fileHeap struct {
	items []File
	worse worseFunc
}

func (h *fileHeap) Len() int           { return len(h.items) }
func (h *fileHeap) Less(i, j int) bool { return h.worse(h.items[i], h.items[j]) }
func (h *fileHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *fileHeap) Push(x any) { h.items = append(h.items, x.(File)) }

func (h *fileHeap) Pop() any {
	old := h.items
	n := len(old)
	last := old[n-1]
	h.items = old[:n-1]
	return last
}

// topN keeps the best `limit` files it is shown, in memory bounded by limit
// rather than by how many files exist.
//
// This is the difference between holding 871,018 File structs to report 15 of
// them and holding 15. On a whole-volume scan the old approach cost well over a
// hundred megabytes for a result you could fit on one screen.
//
// A topN is not safe for concurrent use. The scan gives each worker its own and
// merges them at the end, which keeps the hot path lock-free.
type topN struct {
	limit int
	h     *fileHeap
}

func newTopN(limit int, worse worseFunc) *topN {
	if limit < 0 {
		limit = 0
	}
	return &topN{
		limit: limit,
		h:     &fileHeap{worse: worse, items: make([]File, 0, min(limit, 1024))},
	}
}

// add offers a file. It is kept only if there is room or it beats the worst
// entry currently held.
func (t *topN) add(f File) {
	if t.limit == 0 {
		return
	}

	if t.h.Len() < t.limit {
		heap.Push(t.h, f)
		return
	}

	// The root is the weakest entry held. If the candidate is no better, it
	// cannot belong in the result and is dropped without further work.
	if t.h.worse(t.h.items[0], f) {
		t.h.items[0] = f
		heap.Fix(t.h, 0)
	}
}

// merge folds another topN in, so per-worker results can be combined without
// either side having held more than limit entries.
func (t *topN) merge(other *topN) {
	if other == nil {
		return
	}
	for _, f := range other.h.items {
		t.add(f)
	}
}

// sorted returns the held files best first. The heap itself is only partially
// ordered, so this is where the final ordering is imposed.
func (t *topN) sorted() []File {
	out := make([]File, len(t.h.items))
	copy(out, t.h.items)

	sort.Slice(out, func(i, j int) bool {
		// Best first: i precedes j when j is the worse of the two.
		return t.h.worse(out[j], out[i])
	})

	return out
}
