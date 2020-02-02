package concurrent

import (
	"sync/atomic"
	"unsafe"
)

/*
	lock free linked queue.
	ref:
		1. http://ddrv.cn/a/591069
		2. https://coolshell.cn/articles/8239.html
*/

type ConcurrentLinkedQueueNode struct {
	Value interface{}
	Next  *ConcurrentLinkedQueueNode
}

func (node *ConcurrentLinkedQueueNode) casNext(oldV, newV *ConcurrentLinkedQueueNode) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&node.Next)),
		unsafe.Pointer(oldV),
		unsafe.Pointer(newV),
	)
}

type ConcurrentLinkedQueue struct {
	head *ConcurrentLinkedQueueNode
	tail *ConcurrentLinkedQueueNode
	size int64
}

var expunged = unsafe.Pointer(new(interface{}))

func NewConcurrentLinkedQueue() *ConcurrentLinkedQueue {
	dummy := &ConcurrentLinkedQueueNode{}
	dummy.Value = nil
	dummy.Next = nil
	return &ConcurrentLinkedQueue{ // like container/list, use same node
		head: dummy,
		tail: dummy,
	}
}

func (queue *ConcurrentLinkedQueue) casTail(oldV, newV *ConcurrentLinkedQueueNode) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&queue.tail)),
		unsafe.Pointer(oldV),
		unsafe.Pointer(newV),
	)
}

func (queue *ConcurrentLinkedQueue) casHead(oldV, newV *ConcurrentLinkedQueueNode) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&queue.head)),
		unsafe.Pointer(oldV),
		unsafe.Pointer(newV),
	)
}

func (queue *ConcurrentLinkedQueue) Enqueue(v interface{}) bool {
	newNode := &ConcurrentLinkedQueueNode{Value: v, Next: nil}
	for {
		tail := queue.tail
		next := tail.Next
		if tail == queue.tail {
			if next == nil {
				if tail.casNext(next, newNode) {
					queue.casTail(tail, newNode)
					atomic.AddInt64(&queue.size, 1)
					return true
				}
			} else {
				queue.casTail(tail, next)
			}
		}
	}
}

func (queue *ConcurrentLinkedQueue) Dequeue() interface{} {
	for {
		h := queue.head
		t := queue.tail
		first := h.Next
		if h == queue.head {
			if h == t {
				if first == nil {
					return nil
				}
				queue.casTail(t, first)
			} else if queue.casHead(h, first) {
				h.casNext(first, nil)
				return first.Value
			}
		}
	}
}
