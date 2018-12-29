/* Copyright 2018 tabjy
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/* This file contains function(s) derived from an Apache 2.0 licensed software,
 * namely github.com/shadowsocks/go-shadowsocks2, and is therefore licensed
 * under Apache 2.0 separately.
 */

package go_shadowsocks2

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/socks"
	"github.com/tabjy/yagl"
	"golang.org/x/net/proxy"
)

var Log yagl.Logger // set by main.go

func logf(f string, v ...interface{}) {
	Log.Infof(f, v...)
}

type Account interface {
	IncrementTraffic(int64)
	SigChan() chan int
}

// Listen on addr for incoming connections.
// Modified from github.com/shadowsocks/go-shadowsocks2/tcp.go
func TcpRemote(addr string, shadow func(net.Conn) net.Conn, socks5Addr string, account Account) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return
	}

	go func() {
		<-account.SigChan()
		l.Close()
	}()

	logf("listening TCP on %s", addr)
	for {
		c, err := l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logf("failed to accept: %v", err)
			continue
		}

		go func() {
			defer c.Close()
			c.(*net.TCPConn).SetKeepAlive(true)
			c = shadow(c)

			tgt, err := socks.ReadAddr(c)
			if err != nil {
				logf("failed to get target address: %v", err)
				return
			}

			socksDialer, err := proxy.SOCKS5("tcp", socks5Addr, nil, &net.Dialer{})
			if err != nil {
				logf("failed to connect to tor proxy: %v", err)
				return
			}

			rc, err := socksDialer.Dial("tcp", tgt.String())

			if err != nil {
				logf("failed to connect to target: %v", err)
				return
			}
			defer rc.Close()
			rc.(*net.TCPConn).SetKeepAlive(true)

			logf("proxy %s <-> %s", c.RemoteAddr(), tgt)
			down, up, err := relay(c, rc)
			account.IncrementTraffic(up)
			account.IncrementTraffic(down)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				logf("relay error: %v", err)
			}
		}()
	}
}

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
// Taken from github.com/shadowsocks/go-shadowsocks2/tcp.go
func relay(left, right net.Conn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
		ch <- res{n, err}
	}()

	n, err := io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}
