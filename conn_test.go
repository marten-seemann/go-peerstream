package peerstream

import (
	"net"
	"sync"
	"testing"

	tpt "github.com/libp2p/go-libp2p-transport"
	smux "github.com/libp2p/go-stream-muxer"
)

type fakeconn struct {
	tpt.Conn
}

func (f *fakeconn) Close() error {
	return nil
}

var _ net.Conn = new(fakeconn)

type fakeSmuxConn struct {
	smux.Conn
	closed bool
}

func (fsc *fakeSmuxConn) IsClosed() bool {
	return fsc.closed
}

func (fsc *fakeSmuxConn) Close() error {
	return nil
}

var _ smux.Conn = new(fakeSmuxConn)

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
	ca.Close()
	if len(s.connByNet) != 0 || len(s.conns) != 0 {
		t.Fatal("leaked connections")
	}
}

func TestConnIdx(t *testing.T) {
	s := NewSwarm(nil)
	c, err := s.AddConn(new(fakeconn))
	if err != nil {
		t.Fatal(err)
	}

	g := "foo"
	g2 := "bar"

	if len(s.ConnsWithGroup(g)) != 0 {
		t.Fatal("should have gotten none")
	}

	c.AddGroup(g)
	if !c.InGroup(g) {
		t.Fatal("should be in the appropriate group")
	}
	if len(s.ConnsWithGroup(g)) != 1 {
		t.Fatal("should have only gotten one")
	}

	c.Close()
	if !c.InGroup(g) {
		t.Fatal("should still be in the appropriate group")
	}
	if len(s.ConnsWithGroup(g)) != 0 {
		t.Fatal("should have gotten none")
	}

	c.AddGroup(g2)
	if !c.InGroup(g2) {
		t.Fatal("should now be in group 2")
	}
	if c.InGroup("bla") {
		t.Fatal("should not be in arbitrary groups")
	}
	if len(s.ConnsWithGroup(g)) != 0 {
		t.Fatal("should still have gotten none")
	}
	if len(s.ConnsWithGroup(g2)) != 0 {
		t.Fatal("should still have gotten none")
	}
	if len(s.connIdx) != 0 {
		t.Fatal("should have an empty index")
	}
	if len(s.conns) != 0 {
		t.Fatal("should not be holding any connections")
	}
}

func TestAddConnWithGroups(t *testing.T) {
	s := NewSwarm(nil)

	g := "foo"
	g2 := "bar"
	g3 := "baz"

	c, err := s.AddConn(new(fakeconn), g, g2)
	if !c.InGroup(g) || !c.InGroup(g2) || c.InGroup(g3) {
		t.Fatal("should be in the appropriate groups")
	}
	if err != nil {
		t.Fatal(err)
	}

	if len(s.ConnsWithGroup(g)) != 1 {
		t.Fatal("should have gotten one")
	}

	if len(s.ConnsWithGroup(g2)) != 1 {
		t.Fatal("should have gotten one")
	}

	if len(s.ConnsWithGroup(g3)) != 0 {
		t.Fatal("should have gotten none")
	}

	c.Close()
	if len(s.ConnsWithGroup(g)) != 0 {
		t.Fatal("should have gotten none")
	}

	if len(s.ConnsWithGroup(g2)) != 0 {
		t.Fatal("should have gotten none")
	}

	if len(s.ConnsWithGroup(g3)) != 0 {
		t.Fatal("should still have gotten none")
	}

	if len(s.connIdx) != 0 {
		t.Fatal("should have an empty index")
	}
	if len(s.conns) != 0 {
		t.Fatal("should not be holding any connections")
	}
}
