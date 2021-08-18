//+build linux

package fasttcp

import (
	"syscall"
)

const (
	// EpollERR EpollErr report errors
	EpollERR = 0x8
	// EpollET set edge trigger
	EpollET = 0x80000000
	// EpollHUP reports close
	EpollHUP = 0x10
	// EpollIN reports input
	EpollIN = 0x1
	// EpollONESHOT set one shot pulling
	EpollONESHOT = 0x40000000
	// EpollRDHUP reports read side close
	EpollRDHUP = 0x2000
)

//ConnEvents are connection pulling events
var ConnEvents uint32 = EpollIN | EpollHUP | EpollERR | EpollRDHUP | EpollONESHOT | EpollET

//FD represents a posix file descriptor
type FD = int

//Conn represents a connection
type Conn FD

//Close impl io.Closer for conn
func (r Conn) Close() error {
	return syscall.Close(int(r))
}

//Fd returns conn fd
func (r Conn) Fd() FD {
	return FD(r)
}

//Read impl io.Reader for conn
func (r Conn) Read(b []byte) (int, error) {
	return syscall.Read(int(r), b)
}

//Write impl io.Writer for conn
func (r Conn) Write(b []byte) (int, error) {
	return syscall.Write(int(r), b)
}

//Addr return the conn addr
func (r Conn) Addr() syscall.Sockaddr {
	addr, err := syscall.Getsockname(int(r))
	if err != nil {
		panic(err)
	}
	return addr
}

//Server represents a server
type Server FD

//Close impl io.Closer for server
func (r Server) Close() error {
	return syscall.Close(int(r))
}

func (r Server) accept() (Conn, syscall.Sockaddr, error) {
	FD, rAddr, err := syscall.Accept(int(r))
	if err != nil {
		return 0, rAddr, err
	}
	err = syscall.SetNonblock(FD, true)
	if err != nil {
		_ = Conn(FD).Close()
		return 0, rAddr, err
	}
	return Conn(FD), rAddr, nil
}

//LocalAddr returns server listening addr
func (r Server) LocalAddr() syscall.Sockaddr {
	addr, err := syscall.Getsockname(int(r))
	if err != nil {
		panic(err)
	}
	return addr
}

//Fd returns server listener socket fd
func (r Server) Fd() FD {
	return int(r)
}

//EventConn represents a new connection event
type EventConn struct {
	conn   Conn
	server Server
	epoll  FD
	addr   syscall.Sockaddr
}

//RemoteAddr returns new connection addr
func (r *EventConn) RemoteAddr() syscall.Sockaddr {
	return r.addr
}

//Conn returns new conn
func (r *EventConn) Conn() Conn {
	return r.conn
}

//Server returns the acceptor server
func (r *EventConn) Server() Server {
	return r.server
}

//Accept accepts the conn to receive data
func (r *EventConn) Accept() error {
	event := &syscall.EpollEvent{
		Fd:     int32(r.conn.Fd()),
		Events: ConnEvents,
	}
	err := syscall.EpollCtl(r.epoll, syscall.EPOLL_CTL_ADD, r.conn.Fd(), event)
	return err
}

//EventData represents a data event
type EventData struct {
	conn   Conn
	server Server
	epoll  FD
}

//Conn returns data conn
func (r *EventData) Conn() Conn {
	return r.conn
}

//Server returns server
func (r *EventData) Server() Server {
	return r.server
}

//Wakeup continue receiving data from conn
func (r *EventData) Wakeup() error {
	event := &syscall.EpollEvent{
		Fd:     int32(r.conn.Fd()),
		Events: ConnEvents,
	}
	err := syscall.EpollCtl(r.epoll, syscall.EPOLL_CTL_MOD, r.conn.Fd(), event)
	return err
}

//EventClose represents a close conn event
type EventClose struct {
	conn   Conn
	server Server
}

//Conn returns closed conn
func (r *EventClose) Conn() syscall.Sockaddr {
	return r.conn.Addr()
}

//Server returns server
func (r *EventClose) Server() Server {
	return r.server
}

//New make a new server that will listen to addr
func New(addr syscall.Sockaddr) (Server, error) {
	var sock int
	var err error
	switch addr.(type) {
	case *syscall.SockaddrInet6:
		sock, err = syscall.Socket(syscall.AF_INET6, syscall.SOCK_STREAM, 0)
		if err != nil {
			return Server(sock), err
		}
	default:
		sock, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		if err != nil {
			return Server(sock), err
		}
	}
	err = syscall.SetNonblock(sock, true)
	if err != nil {
		Server(sock).Close()
		return Server(sock), err
	}
	err = syscall.Bind(sock, addr)
	if err != nil {
		Server(sock).Close()
		return Server(sock), err
	}
	err = syscall.Listen(sock, 0)
	if err != nil {
		Server(sock).Close()
		return Server(sock), err
	}
	return Server(sock), nil
}

//Listen start pulling the server events and pushing to Channel
func (r Server) Listen(Channel chan interface{}) error {
	return listen(r, Channel)
}

func listen(sock Server, C chan interface{}) (err error) {
	epoll, err := syscall.EpollCreate1(0)
	if err != nil {
		return err
	}
	defer syscall.Close(epoll)
	event := &syscall.EpollEvent{
		Events: EpollIN | EpollHUP | EpollERR | EpollRDHUP,
		Fd:     int32(sock),
	}
	err = syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, int(sock), event)
	if err != nil {
		return err
	}
	events := make([]syscall.EpollEvent, cap(C))
	for {
		nEvents, err := syscall.EpollWait(epoll, events, -1)
		if err != nil {
			if err == syscall.EAGAIN {
				continue
			}
			return err
		}

		for _, event := range events[:nEvents] {
			if event.Events&(EpollHUP|EpollERR|EpollRDHUP) != 0 {
				if int(event.Fd) == int(sock) {
					C <- syscall.EBADF
					return syscall.EBADF
				}
				//syscall.EpollCtl(epoll, syscall.EPOLL_CTL_DEL, int(event.Fd), nil)
				C <- &EventClose{
					conn:   Conn(event.Fd),
					server: sock,
				}
				continue
			}
			if event.Events&(EpollIN|EpollET) != 0 {
				if int(event.Fd) == int(sock) {
					Conn, rAddr, err := Server(event.Fd).accept()
					if err != nil {
						_ = syscall.Close(int(Conn))
						continue
					}
					C <- &EventConn{
						conn:   Conn,
						server: sock,
						epoll:  epoll,
						addr:   rAddr,
					}
				} else {
					C <- &EventData{
						conn:   Conn(event.Fd),
						server: sock,
						epoll:  epoll,
					}
				}
			}

		}
	}
}
