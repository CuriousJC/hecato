package files

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/curiousjc/hecato/internal/ignore"
)

// DefaultWorkers is how many goroutines read directories concurrently.
//
// Eight, measured rather than guessed: on a 12-CPU machine a full c:/ scan took
// 8.9s at 4 workers, 6.3s at 8 and 6.0s at 16. Past 8 the curve flattens, and
// the work is waiting on the filesystem rather than on CPU, so more goroutines
// mostly add contention.
const DefaultWorkers = 8

// dirQueue is an unbounded LIFO queue of directories still to read.
//
// Unbounded on purpose. A buffered channel deadlocks here: workers are also
// producers, so once the buffer fills, every worker blocks trying to enqueue a
// subdirectory and nobody is left to drain it. Growing a slice under a mutex
// avoids that without the unbounded-goroutine trick of spawning a sender per
// blocked push.
//
// LIFO rather than FIFO so the walk stays depth-first-ish, which keeps the
// queue short and the directory entries being read close together on disk.
type dirQueue struct {
	mu   sync.Mutex
	cond *sync.Cond

	items []string

	// pending counts directories handed out or waiting, not yet finished. The
	// walk is complete when it reaches zero, which is the only reliable
	// termination signal: an empty queue means nothing while a worker is still
	// running and might enqueue children.
	pending int

	done bool
}

func newDirQueue() *dirQueue {
	q := &dirQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *dirQueue) push(dir string) {
	q.mu.Lock()
	q.items = append(q.items, dir)
	q.pending++
	q.mu.Unlock()

	q.cond.Signal()
}

// pop blocks until a directory is available, or returns false once the whole
// walk has finished.
func (q *dirQueue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 && !q.done {
		q.cond.Wait()
	}

	if len(q.items) == 0 {
		return "", false
	}

	last := len(q.items) - 1
	dir := q.items[last]
	q.items = q.items[:last]

	return dir, true
}

// finish marks one directory as fully processed. Callers must push a
// directory's children before calling this, or pending can reach zero while
// work remains and the walk will end early.
func (q *dirQueue) finish() {
	q.mu.Lock()
	q.pending--
	ended := q.pending == 0
	if ended {
		q.done = true
	}
	q.mu.Unlock()

	if ended {
		q.cond.Broadcast()
	}
}

// shard is one worker's private accumulator. Nothing here is shared, so the
// scanning path takes no locks at all; the shards are merged once at the end.
type shard struct {
	top        *topN
	errors     []File
	scanned    int
	ignored    int
	pruned     int
	matched    int
	totalBytes int64
}

// scan walks target concurrently and returns the best `limit` files under the
// given ordering, along with the counts describing the walk.
//
// A nil or empty matcher means no filtering. Directories matching the matcher
// are never enqueued, which is how pruning works here -- there is no
// fs.SkipDir equivalent to return, and none is needed: not enqueueing a
// directory is exactly "do not descend into it".
func scan(target string, ig *ignore.Matcher, workers, limit int, worse worseFunc) (*Result, error) {
	started := time.Now()

	if workers < 1 {
		workers = DefaultWorkers
	}
	if workers > runtime.NumCPU()*4 {
		// Guard against a config typo turning into tens of thousands of
		// goroutines all contending for the same disk.
		workers = runtime.NumCPU() * 4
	}

	q := newDirQueue()
	shards := make([]*shard, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		s := &shard{top: newTopN(limit, worse)}
		shards[i] = s

		wg.Add(1)
		go func(s *shard) {
			defer wg.Done()
			for {
				dir, ok := q.pop()
				if !ok {
					return
				}
				readDir(q, s, dir, ig, worse)
				q.finish()
			}
		}(s)
	}

	// The root is enqueued without a MatchDir check. Ignoring the directory you
	// were asked to scan would silently return nothing, so it is exempt.
	q.push(filepath.Clean(target))
	wg.Wait()

	res := &Result{top: newTopN(limit, worse)}
	for _, s := range shards {
		res.top.merge(s.top)
		res.Errors = append(res.Errors, s.errors...)
		res.Scanned += s.scanned
		res.Ignored += s.ignored
		res.Pruned += s.pruned
		res.Matched += s.matched
		res.TotalBytes += s.totalBytes
	}

	res.Files = res.top.sorted()
	res.Elapsed = time.Since(started)

	return res, nil
}

// readDir processes a single directory: subdirectories go back on the queue,
// files go through the ignore rules and into this worker's top-N.
func readDir(q *dirQueue, s *shard, dir string, ig *ignore.Matcher, worse worseFunc) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Collected rather than fatal, so one unreadable directory does not
		// abort a scan of c:/.
		s.errors = append(s.errors, File{Path: dir})
		return
	}

	for _, e := range entries {
		path := filepath.Join(dir, e.Name())

		if e.IsDir() {
			if ig.MatchDir(path) {
				s.pruned++
				continue
			}
			q.push(path)
			continue
		}

		s.scanned++

		if ig.MatchFile(path) {
			s.ignored++
			continue
		}

		// Deferred until after the ignore check, so an ignored file costs a
		// pattern match and nothing else. Info can still fail: a DirEntry may
		// be stale by the time it is read, if the file was deleted between the
		// directory read and this call.
		info, err := e.Info()
		if err != nil {
			s.errors = append(s.errors, File{Path: path})
			continue
		}

		s.matched++
		s.totalBytes += info.Size()
		s.top.add(File{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
}
