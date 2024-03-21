package client

import (
	"net"
	"time"
)

type connectionPool struct {
	url string
	// conn           net.Conn
	chanConnGet    chan chan net.Conn
	chanConnReturn chan connReturn
	chanDrainPool  chan int
}

func newConnectionPool(url string, connectionPoolSize int, waitTimeout time.Duration) *connectionPool {
	connPool := &connectionPool{
		url:            url,
		chanConnGet:    make(chan chan net.Conn),
		chanConnReturn: make(chan connReturn),
		chanDrainPool:  make(chan int),
	}
	go connPool.run(connectionPoolSize, waitTimeout)
	return connPool
}

func (c *connectionPool) run(connectionPoolSize int, waitTimeout time.Duration) {
	type poolEntry struct {
		busy bool
		err  error
		conn net.Conn
	}
	type waitPoolEntry struct {
		entryTime time.Time
		chanConn  chan net.Conn
	}

	var (
		connectionPool = make(map[int]*poolEntry, connectionPoolSize)
		waitPool       = map[int]*waitPoolEntry{}
	)
	for i := 0; i < connectionPoolSize; i++ {
		connectionPool[i] = &poolEntry{
			conn: nil,
			busy: false,
		}
	}
RunLoop:
	for {
		// fmt.Println("----------------------- run loop ------------------------")
		select {
		case <-c.chanDrainPool:
			// fmt.Println("<-c.chanDrainPool")
			for _, waitPoolEntry := range waitPool {
				waitPoolEntry.chanConn <- nil
			}
			break RunLoop
		case <-time.After(waitTimeout):
		//	fmt.Println("tick", len(connectionPool), len(waitPool))
		// 	for i, poolEntry := range connectionPool {
		// 		fmt.Println(i, poolEntry)
		// 	}
		// 	for i, waitPoolEntry := range waitPool {
		// 		fmt.Println(i, waitPoolEntry)
		// 	}
		case chanReturnNextConn := <-c.chanConnGet:
			// fmt.Println("chanReturnNextConn := <-c.chanConnGet:")
			nextI := 0
			for i := range waitPool {
				if i >= nextI {
					nextI = i + 1
				}
			}
			waitPool[nextI] = &waitPoolEntry{
				chanConn:  chanReturnNextConn,
				entryTime: time.Now(),
			}
			// fmt.Println("sbdy wants a new conn", nextI)
		case connReturn := <-c.chanConnReturn:
			// fmt.Println("connReturn := <-c.chanConnReturn:")
			for _, poolEntry := range connectionPool {
				if connReturn.conn == poolEntry.conn {
					poolEntry.busy = false
					if connReturn.err != nil {
						poolEntry.err = connReturn.err
						poolEntry.conn.Close()
						poolEntry.conn = nil
					}
				}
			}
		}
		// refill connection pool
		for _, poolEntry := range connectionPool {
			if poolEntry.conn == nil {
				newConn, errDial := net.Dial("tcp", c.url)
				poolEntry.err = errDial
				poolEntry.conn = newConn
			}
		}
		// redistribute available connections
		for _, poolEntry := range connectionPool {
			if len(waitPool) == 0 {
				break
			}
			if poolEntry.err == nil && poolEntry.conn != nil && !poolEntry.busy {
				for i, waitPoolEntry := range waitPool {
					// fmt.Println("---------------------------> serving wait pool", i, waitPoolEntry)
					poolEntry.busy = true
					delete(waitPool, i)
					waitPoolEntry.chanConn <- poolEntry.conn
					break
				}
			}
		}
		// waitpool cleanup
		var (
			waitPoolLoosers = []int{}
			now             = time.Now()
		)
		for i, waitPoolEntry := range waitPool {
			if now.Sub(waitPoolEntry.entryTime) > waitTimeout {
				waitPoolLoosers = append(waitPoolLoosers, i)
				waitPoolEntry.chanConn <- nil
			}
		}
		for _, i := range waitPoolLoosers {
			delete(waitPool, i)
		}

	}
	c.chanDrainPool = nil
	c.chanConnReturn = nil
	c.chanConnGet = nil
	// fmt.Println("runloop is done", waitPool)
}
