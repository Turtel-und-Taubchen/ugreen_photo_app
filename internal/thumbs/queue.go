package thumbs

// job is one preview waiting for or undergoing a render. It is identified by id
// (cache key + size); the service keeps at most one job per id.
type job struct {
	id    string
	entry Entry
	size  string
	dst   string
	// high and stamp are owned by queue: a job sits in the high stack once per request,
	// and only the slot carrying its latest stamp is live.
	high  bool
	stamp uint64
}

type slot struct {
	j     *job
	stamp uint64
}

// queue orders waiting jobs: requested (high) jobs before background (low) ones, the most
// recently requested high job first — it is the one on screen — and low jobs in arrival
// order. Promoting or re-requesting a job does not search the slices; it pushes a new
// slot and pop skips the stale ones, so a library-sized backlog costs nothing per request.
// Not safe for concurrent use; the service guards it with its mutex.
type queue struct {
	high  []slot
	low   []*job
	stamp uint64
}

// pushHigh makes j the next job to run, whether it is new, waiting as a background job
// or already waiting as a requested one.
func (q *queue) pushHigh(j *job) {
	q.stamp++
	j.high = true
	j.stamp = q.stamp
	q.high = append(q.high, slot{j: j, stamp: q.stamp})
}

func (q *queue) pushLow(j *job) {
	q.low = append(q.low, j)
}

// pop removes and returns the next job, or nil when none is waiting.
func (q *queue) pop() *job {
	for n := len(q.high); n > 0; n = len(q.high) {
		s := q.high[n-1]
		q.high[n-1] = slot{}
		q.high = q.high[:n-1]
		if s.j.stamp == s.stamp {
			return s.j
		}
	}
	q.high = nil
	for len(q.low) > 0 {
		j := q.low[0]
		q.low[0] = nil
		q.low = q.low[1:]
		// A promoted job left its background slot behind; it runs from the high stack.
		if !j.high {
			return j
		}
	}
	q.low = nil
	return nil
}
