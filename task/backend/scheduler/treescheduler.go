package scheduler

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/cespare/xxhash"
	"github.com/google/btree"
)

const (
	// degreeBtreeScheduled is the btree degree for the btree internal to the tree scheduler.
	// it is purely a performance tuning parameter, but required by github.com/google/btree
	degreeBtreeScheduled = 3 // TODO(docmerlin): find the best number for this, its purely a perf optimization

	// defaultMaxWorkers is a constant that sets the default number of maximum workers for a TreeScheduler
	defaultMaxWorkers = 128
)

// TreeScheduler is a Scheduler based on a btree.
// It calls Executor in-order per ID.  That means you are guaranteed that for a specific ID,
//
// - The scheduler should, after creation, automatically call ExecutorFunc, when a task should run as defined by its Schedulable.
//
// - the scheduler's should not be able to get into a state where blocks Release and Schedule indefinitely.
//
// - Schedule should add a Schedulable to being scheduled, and Release should remove a task from being scheduled.
//
// - Calling of ExecutorFunc should be serial in time on a per taskID basis. I.E.: the run at 12:00 will go before the run at 12:01.
//
// Design:
//
// The core of the scheduler is a btree keyed by time, a nonce, and a task ID, and a map keyed by task ID and containing a
// nonce and a time (called a uniqueness index from now on).
// The map is to ensure task uniqueness in the tree, so we can replace or delete tasks in the tree.
//
// Scheduling in the tree consists of a main loop that feeds a fixed set of workers, each with their own communication channel.
// Distribution is handled by hashing the TaskID (to ensure uniform distribution) and then distributing over those channels
// evenly based on the hashed ID.  This is to ensure that all tasks of the same ID go to the same worker.
//
//The workers call ExecutorFunc handle any errors and update the LastScheduled time internally and also via the Checkpointer.
//
// The main loop:
//
// The main loop waits on a time.Timer to grab the task with the minimum time.  Once it successfully grabs a task ready
// to trigger, it will start walking the btree from the item nearest
//
// Putting a task into the scheduler:
//
// Adding a task to the scheduler acquires a write lock, grabs the task from the uniqueness map, and replaces the item
// in the uniqueness index and btree.  If new task would trigger sooner than the current soonest triggering task, it
// replaces the Timer when added to the scheduler.  Finally it releases the write lock.
//
// Removing a task from the scheduler:
//
// Removing a task from the scheduler acquires a write lock, deletes the task from the uniqueness index and from the
// btree, then releases the lock.  We do not have to readjust the time on delete, because, if the minimum task isn't
// ready yet, the main loop just resets the timer and keeps going.
type TreeScheduler struct {
	mu           sync.RWMutex
	scheduled    *btree.BTree
	nextTime     map[ID]ordering // we need this index so we can delete items from the scheduled
	when         time.Time
	executor     Executor
	onErr        ErrorFunc
	time         clock.Clock
	timer        *clock.Timer
	done         chan struct{}
	workchans    []chan Item
	wg           sync.WaitGroup
	checkpointer SchedulableService

	sm *SchedulerMetrics
}

// ErrorFunc is a function for error handling.  It is a good way to inject logging into a TreeScheduler.
type ErrorFunc func(ctx context.Context, taskID ID, scheduledAt time.Time, err error)

type treeSchedulerOptFunc func(t *TreeScheduler) error

// WithOnErrorFn is an option that sets the error function that gets called when there is an error in a TreeScheduler.
// its useful for injecting logging or special error handling.
func WithOnErrorFn(fn ErrorFunc) treeSchedulerOptFunc {
	return func(t *TreeScheduler) error {
		t.onErr = fn
		return nil
	}
}

// WithMaxConcurrentWorkers is an option that sets the max number of concurrent workers that a TreeScheduler will use.
func WithMaxConcurrentWorkers(n int) treeSchedulerOptFunc {
	return func(t *TreeScheduler) error {
		t.workchans = make([]chan Item, n)
		return nil
	}
}

// WithTime is an optiom for NewScheduler that allows you to inject a clock.Clock from ben johnson's github.com/benbjohnson/clock library, for testing purposes.
func WithTime(t clock.Clock) treeSchedulerOptFunc {
	return func(sch *TreeScheduler) error {
		sch.time = t
		return nil
	}
}

// NewScheduler gives us a new TreeScheduler and SchedulerMetrics when given an  Executor, a SchedulableService, and zero or more options.
// Schedulers should be initialized with this function.
func NewScheduler(executor Executor, checkpointer SchedulableService, opts ...treeSchedulerOptFunc) (*TreeScheduler, *SchedulerMetrics, error) {
	s := &TreeScheduler{
		executor:     executor,
		scheduled:    btree.New(degreeBtreeScheduled),
		nextTime:     map[ID]ordering{},
		onErr:        func(_ context.Context, _ ID, _ time.Time, _ error) {},
		time:         clock.New(),
		done:         make(chan struct{}, 1),
		checkpointer: checkpointer,
	}

	// apply options
	for i := range opts {
		if err := opts[i](s); err != nil {
			return nil, nil, err
		}
	}
	if s.workchans == nil {
		s.workchans = make([]chan Item, defaultMaxWorkers)

	}

	s.wg.Add(len(s.workchans))
	for i := 0; i < len(s.workchans); i++ {
		s.workchans[i] = make(chan Item)
		go s.work(context.Background(), s.workchans[i])
	}

	s.sm = NewSchedulerMetrics(s)
	s.when = time.Time{}
	s.timer = s.time.Timer(0)
	s.timer.Stop()
	// because a stopped timer will wait forever, this allows us to wait for items to be added before triggering.

	if executor == nil {
		return nil, nil, errors.New("executor must be a non-nil function")
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
	schedulerLoop:
		for {
			select {
			case <-s.done:
				s.mu.Lock()
				s.timer.Stop()
				// close workchans
				for i := range s.workchans {
					close(s.workchans[i])
				}
				s.mu.Unlock()
				return
			case <-s.timer.C:
				for { // this for loop is a work around to the way clock's mock works when you reset duration 0 in a different thread than you are calling your clock.Set
					s.mu.Lock()
					min := s.scheduled.Min()
					if min == nil { // grab a new item, because there could be a different item at the top of the queue
						s.when = time.Time{}
						s.mu.Unlock()
						continue schedulerLoop
					}
					it := min.(Item)
					if ts := s.time.Now().UTC(); it.when().After(ts) {
						s.timer.Reset(ts.Sub(it.when()))
						s.mu.Unlock()
						continue schedulerLoop
					}
					s.process()
					min = s.scheduled.Min()
					if min == nil { // grab a new item, because there could be a different item at the top of the queue after processing
						s.when = time.Time{}
						s.mu.Unlock()
						continue schedulerLoop
					}
					it = min.(Item)
					s.when = it.when()
					until := s.when.Sub(s.time.Now())

					if until > 0 {
						s.resetTimer(until) // we can reset without a stop because we know it is fired here
						s.mu.Unlock()
						continue schedulerLoop
					}
					s.mu.Unlock()
				}
			}
		}
	}()
	return s, s.sm, nil
}

func (s *TreeScheduler) Stop() {
	s.mu.Lock()
	close(s.done)
	s.mu.Unlock()
	s.wg.Wait()
}

type unsent struct {
	items []Item
}

func (u *unsent) append(i Item) {
	u.items = append(u.items, i)
}

func (s *TreeScheduler) process() {
	iter, toReAdd := s.iterator(s.time.Now())
	s.scheduled.Ascend(iter)
	for i := range toReAdd.items {
		s.nextTime[toReAdd.items[i].id] = toReAdd.items[i].ordering
		s.scheduled.ReplaceOrInsert(toReAdd.items[i])
	}
}

func (s *TreeScheduler) resetTimer(whenFromNow time.Duration) {
	s.when = s.time.Now().Add(whenFromNow)
	s.timer.Reset(whenFromNow)
}

func (s *TreeScheduler) iterator(ts time.Time) (btree.ItemIterator, *unsent) {
	itemsToPlace := &unsent{}
	return func(i btree.Item) bool {
		if i == nil {
			return false
		}
		it := i.(Item) // we want it to panic if things other than Items are populating the scheduler, as it is something we can't recover from.
		if time.Unix(it.next+it.Offset, 0).After(ts) {
			return false
		}
		// distribute to the right worker.
		{
			buf := [8]byte{}
			binary.LittleEndian.PutUint64(buf[:], uint64(it.id))
			wc := xxhash.Sum64(buf[:]) % uint64(len(s.workchans)) // we just hash so that the number is uniformly distributed
			select {
			case s.workchans[wc] <- it:
				s.scheduled.Delete(it)
				if err := it.updateNext(); err != nil {
					// in this error case we can't schedule next, so we have to drop the task
					s.onErr(context.Background(), it.id, it.Next(), &ErrUnrecoverable{err})
					return true
				}
				itemsToPlace.append(it)

			case <-s.done:
				return false
			default:
				s.scheduled.Delete(it)
				it.incrementNonce()
				itemsToPlace.append(it)
				return true
			}
		}
		return true
	}, itemsToPlace
}

// When gives us the next time the scheduler will run a task.
func (s *TreeScheduler) When() time.Time {
	s.mu.RLock()
	w := s.when
	s.mu.RUnlock()
	return w
}

func (s *TreeScheduler) release(taskID ID) {
	ordering, ok := s.nextTime[taskID]
	if !ok {
		return
	}

	// delete the old task run time
	s.scheduled.Delete(Item{id: taskID, ordering: ordering})
	delete(s.nextTime, taskID)
}

// Release releases a task.
// Release also cancels the running task.
// Task deletion would be faster if the tree supported deleting ranges.
func (s *TreeScheduler) Release(taskID ID) {
	s.sm.release(taskID)
	s.mu.Lock()
	s.release(taskID)
	s.mu.Unlock()
}

// work does work from the channel and checkpoints it.
func (s *TreeScheduler) work(ctx context.Context, ch chan Item) {
	var it Item
	defer func() {
		s.wg.Done()
	}()
	for it = range ch {
		t := time.Unix(it.next, 0)
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &ErrUnrecoverable{errors.New("executor panicked")}
				}
			}()
			return s.executor.Execute(ctx, it.id, t)
		}()
		if err != nil {
			s.onErr(ctx, it.id, it.Next(), err)
		}
		// TODO(docmerlin): we can increase performance by making the call to UpdateLastScheduled async
		if err := s.checkpointer.UpdateLastScheduled(ctx, it.id, t); err != nil {
			s.onErr(ctx, it.id, it.Next(), err)
		}
	}
}

// Schedule put puts a Schedulable on the TreeScheduler.
func (s *TreeScheduler) Schedule(sch Schedulable) error {
	s.sm.schedule(sch.ID())
	it := Item{
		cron:   sch.Schedule(),
		id:     sch.ID(),
		Offset: int64(sch.Offset().Seconds()),
		//last:   sch.LastScheduled().Unix(),
	}
	nt, err := it.cron.Next(sch.LastScheduled())
	if err != nil {
		return err
	}
	it.next = nt.UTC().Unix()
	it.ordering.when = it.next + it.Offset

	s.mu.Lock()
	defer s.mu.Unlock()

	nt = nt.Add(sch.Offset())
	if s.when.IsZero() || s.when.After(nt) {
		s.when = nt
		s.timer.Stop()
		until := s.when.Sub(s.time.Now())
		if until <= 0 {
			s.timer.Reset(0)
		} else {
			s.timer.Reset(s.when.Sub(s.time.Now()))
		}
	}
	nextTime, ok := s.nextTime[it.id]

	if ok {
		// delete the old task run time
		s.scheduled.Delete(Item{
			ordering: nextTime,
			id:       it.id,
		})
	}
	s.nextTime[it.id] = ordering{when: it.next + it.Offset}

	// insert the new task run time
	s.scheduled.ReplaceOrInsert(it)
	return nil
}

type ordering struct {
	when  int64
	nonce int // for retries
}

func (k *ordering) incrementNonce() {
	k.nonce++
}

// Item is a task in the scheduler.
type Item struct {
	ordering
	id     ID
	cron   Schedule
	next   int64
	Offset int64
}

func (it Item) Next() time.Time {
	return time.Unix(it.next, 0)
}

func (it Item) when() time.Time {
	return time.Unix(it.ordering.when, 0)
}

// Less tells us if one Item is less than another
func (it Item) Less(bItem btree.Item) bool {
	it2 := bItem.(Item)
	return it.ordering.when < it2.ordering.when || (it.ordering.when == it2.ordering.when && (it.nonce < it2.nonce || it.nonce == it2.nonce && it.id < it2.id))
}

func (it *Item) updateNext() error {
	newNext, err := it.cron.Next(time.Unix(it.next, 0))
	if err != nil {
		return err
	}
	it.next = newNext.UTC().Unix()
	it.ordering.when = it.next + it.Offset
	return nil
}
