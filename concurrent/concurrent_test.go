package concurrent

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestGlobalSingle(t *testing.T) {
	n := runtime.NumCPU()
	fmt.Println("cup num=", n)
	runtime.GOMAXPROCS(n)
	fmt.Println("start TestGlobalSingle")
	c := make(chan int64, 5)
	for i := 0; i < 20; i++ {
		go func() {
			s := GetGlobalSingle()
			fmt.Println("send value", s.Value)
			c <- s.Value
			fmt.Println("finish send ", s.Value)
		}()
	}

	/* NOTE: 如果不另外开一个goroutine来消费channel的话，程序会panic，报错是所有goroutine都休眠了
	具体原因涉及到channel的for range和goroutine的调度机制。
	1. for range遍历channel如何判断已经消费到了最后一个？实际上for range等同于 for{v,ok=<-c if!ok{break}}
	其中ok用来标识channel是否关闭，如果没有关闭channel，那么消费的g就会一直堵塞等待消息，也就是asleep状态
	2. 如果没有新开一个g，那么就是主线程一直陷入堵塞状态；如果新开了一个g的话，其实这个g就泄露了，因为他一直在等待channel的消息；
	而channel中还有等待的g，那这个channel也不会被垃圾回收掉
	*/
	go func() {
		for v := range c {
			fmt.Println("receive value ", v)
		}
	}()
	fmt.Println("end")
	time.Sleep(3 * time.Second)
}

// 新开两个子线程，分别输出1,3,5,7,9...和2,4,6,8,10...，主线程接受子线程的值，输出1,2,3,4,5...
func TestAlternateOutput(t *testing.T) {
	n := runtime.NumCPU()
	fmt.Println("cpu num=", n)
	runtime.GOMAXPROCS(n)

	fmt.Println("====== start ======")
	fmt.Println("stage 0, go num=", runtime.NumGoroutine()) // 默认两个go

	// AlternateOutputViaChannel()
	// AlternateOutputViaAtomic()
	AlternateOutputViaCond()

	fmt.Println("====== end ======")

}

func TestLinkedQueue(t *testing.T) {
	num := runtime.NumCPU()
	runtime.GOMAXPROCS(num)
	wg := sync.WaitGroup{}
	q := NewLinkedQueue()
	concurrency := 10 // 并发
	iterations := 200 // 单个并发执行数量
	wg.Add(concurrency)
	for x := 0; x < concurrency; x++ {
		go func() {
			for i := 0; i < iterations; i++ {
				q.Enqueue(i)
				//q.EnqueueWithLock(i)
			}
			wg.Done()
		}()
	}

	wg.Add(concurrency)
	for x := 0; x < concurrency; x++ {
		go func() {
			i := 0
			for i < iterations {
				if q.Dequeue() != nil {
				//if q.DequeueWithLock() != nil {
					i++
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("cup num=", num)
	fmt.Println("==== end ====")
}

func BenchmarkLinkedQueue(b *testing.B) {
	concurrency := 10
	q := NewLinkedQueue()
	wg := sync.WaitGroup{}
	iterations := b.N
	b.ReportAllocs()
	b.ResetTimer()

	wg.Add(concurrency)
	for x := 0; x < concurrency; x++ {
		go func() {
			for i := 0; i < iterations; i++ {
				//q.Enqueue(i)
				q.EnqueueWithLock(i)
			}
			wg.Done()
		}()
	}

	wg.Add(concurrency)
	for x := 0; x < concurrency; x++ {
		go func() {
			i := 0
			for i < iterations {
				//if q.Dequeue() != nil {
				if q.DequeueWithLock() != nil {
					i++
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkChannel(b *testing.B) {
	iterations := int64(b.N)
	b.ReportAllocs()
	b.ResetTimer()

	channel := make(chan int64, 10000)
	writers := int64(5)

	for x := int64(0); x < writers; x++ {
		go func() {
			for i := int64(0); i < iterations; i++ {
				channel <- i
			}
		}()
	}

	for i := int64(0); i < iterations*writers; i++ {
		<-channel
	}
}

func TestCacheLine(t *testing.T) {
	arr := [1024 * 1024][8]int64{}
	for i := 0; i < 1024*1024; i++ {
		for j := 0; j < 8; j++ {
			arr[i][j] = int64(1)
		}
	}

	sum := int64(0)
	t1 := time.Now()
	for i := 0; i < 1024*1024; i++ {
		tmp := arr[i]
		for j := 0; j < 8; j++ {
			sum = tmp[j]
		}
	}
	fmt.Println("1 cost", time.Since(t1))

	t2 := time.Now()
	for i := 0; i < 8; i++ {
		for j := 0; j < 1024*1024; j++ {
			sum = arr[j][i]
		}
	}
	fmt.Println("2 cost", time.Since(t2))
	fmt.Println(sum)
}

func TestArrayQueue(t *testing.T) {
	num := runtime.NumCPU()
	runtime.GOMAXPROCS(num)

	//wg := sync.WaitGroup{}
	c := make(chan []int, 1000)
	q := NewArrayQueue()
	concurrency := 10 // 并发
	iterations := 20 // 单个并发执行数量

	//wg.Add(concurrency)
	for n := 0; n < concurrency; n++ {
		go func(n int) {
			for i := 0; i < iterations; i++ {
				q.Enqueue([]int{n, i})
				//q.EnqueueWithLock(i)
			}
			//wg.Done()
		}(n)
	}

	//wg.Add(concurrency)
	for n := 0; n < concurrency; n++ {
		go func() {
			i := 0
			for i < iterations {
				if v := q.Dequeue(); v != nil {
					//if q.DequeueWithLock() != nil {
					c <- v.([]int)
					i++
				}
			}
			//wg.Done()
		}()
	}
	//wg.Wait()

	ret := map[int][]int{}
	for i := 0; i < concurrency * iterations ; i++ {
		val := <- c
		if _, ok := ret[val[0]]; ok {
			ret[val[0]] = append(ret[val[0]], val[1])
		} else {
			ret[val[0]] = []int{val[1]}
		}
	}

	for k := range ret {
		fmt.Println(k, ret[k])
	}
	fmt.Println("==== end ====")
}