package skytable_go

import (
	"log"
	"net"

	"github.com/go-baa/pool"
)

func main() {
	// create, initialize cap, max cap, create function
	pl, err := pool.New(2, 10, func() interface{} {
		addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2003")
		cli, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Fatalf("create client connection error: %v\n", err)
		}
		return cli
	})
	if err != nil {
		log.Fatalf("create pool error: %v\n", err)
	}

	pl.Ping = func(conn interface{}) bool {
		// check connection status
		return true
	}

	pl.Close = func(conn interface{}) {
		// close connection
		conn.(*net.TCPConn).Close()
	}

	// get conn from pool
	c, err := pl.Get()
	if err != nil {
		log.Printf("get client error: %v\n", err)
	}
	conn := c.(*net.TCPConn)
	conn.Write([]byte("*1\n~3\n3\nSET\n2\nxx\n3\nexo\n"))
	result := make([]byte, 16)
	n, err := conn.Read(result)
	if err != nil || n < 4 {
		log.Printf("read data error: %v, size: %d\n", err, n)
	}
	log.Printf("got data: %s\n", result)

	// put, back for reuse
	pl.Put(conn)

	// len
	log.Printf("total connections: %d\n", pl.Len())

	// destroy, close all connections
	pl.Destroy()
}
