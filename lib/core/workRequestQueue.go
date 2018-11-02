package core

// queue with limited size, similar to a buffer
import (
	"fmt"
	"sync"

	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
)

const size = 10

type RequestQueue struct {
	sync.Mutex
	//_map map[string]*Task
	array        [size]*CheckoutRequest
	push_element int // points to last (newest) element
	pull_element int // points to first (oldest) element
	count        int
}

func NewRequestQueue() (q *RequestQueue) {
	q = &RequestQueue{push_element: -1, pull_element: -1, count: 0}
	return q
}

func (q *RequestQueue) Push(req *CheckoutRequest) (err error) {
	q.Lock()
	defer q.Unlock()
	logger.Debug(3, "(RequestQueue/Push) %d %d", q.pull_element, q.push_element)
	next_push_element := (q.push_element + 1) % size
	if next_push_element == q.pull_element {
		err = fmt.Errorf(e.QueueFull)
		logger.Debug(3, "(RequestQueue/Push) %s", e.QueueFull)
		return
	}
	q.push_element = next_push_element
	q.array[q.push_element] = req
	if q.pull_element == -1 {
		q.pull_element = 0
	}
	logger.Debug(3, "(RequestQueue/Push) inserted request %d %d", q.pull_element, q.push_element)
	q.count += 1
	return

}

func (q *RequestQueue) Pull() (req *CheckoutRequest, err error) {
	q.Lock()
	defer q.Unlock()
	logger.Debug(3, "(RequestQueue/Pull) %d %d  (count=%d)", q.pull_element, q.push_element, q.count)

	if q.pull_element == -1 {
		err = fmt.Errorf(e.QueueEmpty)
		logger.Debug(3, "(RequestQueue/Pull) %s", e.QueueEmpty)
		return
	}

	req = q.array[q.pull_element]

	next_pull_element := (q.pull_element + 1) % size

	if next_pull_element == q.pull_element {
		q.pull_element = -1
		q.push_element = -1
	}

	logger.Debug(3, "(RequestQueue/Pull) returning request %d %d", q.pull_element, q.push_element)
	return
}
