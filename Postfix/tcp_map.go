/*
	TPC map utility for Postfix. Test it as
	# postmap -q - tcp:127.0.0.1:10044 < /tmp/alist

	Function lookup() should get a key as a string
	and return a correct reply to Postfix type of []byte.

	by aadz, 2017, all rights look as lefts
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	str "strings"
)

var (
	cfgListenOn string
	cfgDebug    bool
)

func lookup(key string) []byte {
	// map a request as "out" string and
	// (reminder of the requests' bytes sum divided by 16) + 1
	var b = []byte(key)
	var sum uint

	for i := range b {
		sum += uint(b[i])
	}

	// build result as a reply to the Postfix query
	result := fmt.Sprintf("200 out%0.2d\n", sum%16+1)
	return []byte(result)
}

func connHandler(conn *net.TCPConn) {
	buf := make([]byte, 256)
	var req string

theHandler:
	for {
		for !str.HasSuffix(req, "\n") {
			cnt, err := conn.Read(buf)
			if err != nil {
				if cfgDebug && err == io.EOF { // connection closed by client
					log.Printf("connection from %v closed", conn.RemoteAddr())
				} else {
					log.Printf("cannot read the request: %v", err)
				}
				break theHandler
			}
			req += string(buf[0:cnt])
		}

		// split the request to a string slice
		reqSlice := str.Split(req[:len(req)-1], " ")
		if reqSlice[0] == "get" { // ignore "put" requests from Postfix
			rep := lookup(reqSlice[1])
			conn.Write(rep)
			if cfgDebug {
				log.Printf("map %s to %s", reqSlice[1], rep)
			}
		}
		req = "" // it is importatnt to set the request string empty here
	}
	conn.Close()
}

func cmdLineGet() {
	flag.StringVar(&cfgListenOn, "l", "localhost:10044", "[address]:port to listen on")
	flag.BoolVar(&cfgDebug, "d", false, "enable debug logging")
	flag.Parse()
	//log.Print("iIt will listen on ", cfgListenOn)
}

func errExit(e error) {
	if e != nil {
		log.Printf("fatal: %s", e)
		os.Exit(1)
	}
}

func main() {
	cmdLineGet()
	lAddr, err := net.ResolveTCPAddr("tcp", cfgListenOn)
	errExit(err)
	in, err := net.ListenTCP("tcp", lAddr)
	errExit(err)
	log.Printf("listening on %v, %v\n", lAddr, in)

	for {
		clientConn, err := in.AcceptTCP()
		if err == nil {
			if cfgDebug {
				log.Printf("clent connection from %v", clientConn.RemoteAddr())
			}
			go connHandler(clientConn)
		} else {
			log.Printf("could not accept client connection: %v", err)
		}
	}
}
