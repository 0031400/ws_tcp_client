package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	tcpAddr := flag.String("tcp", "127.0.0.1:3000", "the tcp listener")
	wsAddr := flag.String("ws", "ws://127.0.0.1:3000", "the ws addr")
	cookie := flag.String("cookie", "", "the ws cookie")
	flag.Parse()
	l, err := net.Listen("tcp", *tcpAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()
	log.Printf("The server is listening on %s\n", *tcpAddr)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("connect from %s\n", c.LocalAddr())
		go handle(c, *wsAddr, *cookie)
	}
}
func handle(tcpConn net.Conn, wsAddr string, cookie string) {
	defer func() {
		tcpConn.Close()
		log.Printf("disconnect from %s\n", tcpConn.LocalAddr())
	}()
	wsConn, _, err := websocket.DefaultDialer.Dial(wsAddr, http.Header{"cookie": []string{cookie}})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("connect ws")
	defer func() {
		wsConn.Close()
		log.Println("disconnect from ws")
	}()
	copy(tcpConn, wsConn)
}

func copy(tcpConn net.Conn, wsConn *websocket.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := wsConn.ReadMessage()
				if err != nil {
					cancel()
					log.Println(err)
					return
				}
				_, err = tcpConn.Write(msg)
				if err != nil {
					cancel()
					log.Println(err)
					return
				}
			}
		}
	}()
	go func() {
		buf := make([]byte, 512)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := tcpConn.Read(buf)
				if err == io.EOF {
					err = wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
					if err != nil {
						cancel()
						log.Println(err)
						return
					}
				} else if err != nil {
					cancel()
					log.Println(err)
					return
				}
				if n > 0 {
					err = wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
					if err != nil {
						cancel()
						log.Println(err)
						return
					}
				}
			}
		}
	}()
	<-ctx.Done()
}
