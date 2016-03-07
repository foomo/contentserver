package server

type stats struct {
	requests  int64
	chanCount chan int
}

func newStats() *stats {
	s := &stats{
		requests:  0,
		chanCount: make(chan int),
	}
	go func() {
		for {
			select {
			case <-s.chanCount:
				s.requests++
				s.chanCount <- 1
			}
		}
	}()
	return s
}

func (s *stats) countRequest() {
	s.chanCount <- 1
	<-s.chanCount
}
