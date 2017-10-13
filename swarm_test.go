package peerstream

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	tpt "github.com/libp2p/go-libp2p-transport"
	smux "github.com/libp2p/go-stream-muxer"
)

type fakeconn struct {
	tpt.Conn
}

func (f *fakeconn) Close() error {
	return nil
}

type fakeTransport struct {
	f func(c net.Conn, isServer bool) (smux.Conn, error)
}

func (f fakeTransport) NewConn(c net.Conn, isServer bool) (smux.Conn, error) {
	return (f.f)(c, isServer)
}

type myNotifee struct {
	conns  map[*Conn]bool
	failed bool
}

func (mn *myNotifee) Connected(c *Conn) {
	_, ok := mn.conns[c]
	if ok {
		fmt.Println("got connected notif for already connected peer")
		mn.failed = true
		return
	}

	mn.conns[c] = true
	time.Sleep(time.Millisecond * 5)
}

func (mn *myNotifee) Disconnected(c *Conn) {
	_, ok := mn.conns[c]
	if !ok {
		fmt.Println("got disconnected notif for unknown peer")
		mn.failed = true
		return
	}

	delete(mn.conns, c)
}

func (mn *myNotifee) OpenedStream(*Stream) {}
func (mn *myNotifee) ClosedStream(*Stream) {}

func TestNotificationOrdering(t *testing.T) {
	s := NewSwarm(nil)
	notifiee := &myNotifee{conns: make(map[*Conn]bool)}

	s.Notify(notifiee)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				nc := new(fakeconn)
				c, err := s.AddConn(nc)
				if err != nil {
					t.Error(err)
				}
				c.Close()
			}
		}()
	}

	wg.Wait()
	if notifiee.failed {
		t.Fatal("we've got problems")
	}
}

type fakeSmuxConn struct {
	smux.Conn
	closed bool
}

func (fsc fakeSmuxConn) IsClosed() bool {
	return fsc.closed
}

func (fsc fakeSmuxConn) Close() error {
	return nil
}

func TestConnsWithGroup(t *testing.T) {
	s := NewSwarm(nil)
	a := newConn(nil, &fakeSmuxConn{}, s)
	b := newConn(nil, &fakeSmuxConn{closed: true}, s)
	c := newConn(nil, &fakeSmuxConn{closed: true}, s)
	g := "foo"

	s.conns[a] = struct{}{}
	s.conns[b] = struct{}{}
	s.conns[c] = struct{}{}

	a.AddGroup(g)
	b.AddGroup(g)
	c.AddGroup(g)

	conns := s.ConnsWithGroup(g)
	if len(conns) != 1 {
		t.Fatal("should have only gotten one")
	}

	if !b.closing {
		t.Fatal("b should at least be closing")
	}

	if !c.closing {
		t.Fatal("c should at least be closing")
	}
}

func TestAddConnTwice(t *testing.T) {
	ready := new(sync.WaitGroup)
	pause := make(chan struct{})
	conns := make(chan *Conn)
	s := NewSwarm(fakeTransport{func(c net.Conn, isServer bool) (smux.Conn, error) {
		ready.Done()
		<-pause
		return nil, nil
	}})
	c := new(fakeconn)
	for i := 0; i < 2; i++ {
		ready.Add(1)
		go func() {
			pc, err := s.AddConn(c)
			if err != nil {
				t.Error(err)
			}
			conns <- pc
		}()
	}
	ready.Wait()
	close(pause)

	ca := <-conns
	cb := <-conns

	if ca != cb {
		t.Fatalf("initialized a single net conn twice: %v != %v", ca, cb)
	}
}
