package fasttcp

import (
	"os"
	"syscall"
	"time"
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

type Server struct {
	fd    fd
	epoll fd
}

func (s *Server) accept() (*Conn, error) {
	f, rAddr, err := syscall.Accept(s.fd.asInt())
	if err != nil {
		return nil, err
	}
	err = syscall.SetNonblock(f, true)
	if err != nil {
		_ = close(f)
		return nil, err
	}
	return &Conn{
		fd:   fd(f),
		addr: rAddr,
	}, nil
}

var ErrAgain = syscall.EAGAIN

func (s *Server) Pull(maxN uint, timeout time.Duration) ([]Event, error) {
	epollEvents := make([]syscall.EpollEvent, maxN)
	nEvents, err := syscall.EpollWait(s.epoll.asInt(), epollEvents, int(timeout.Milliseconds()))
	if err != nil {
		return nil, err
	}
	events := make([]Event, nEvents)
	i := 0
	for _, event := range epollEvents[:nEvents] {
		if event.Events&(EpollHUP|EpollERR|EpollRDHUP) != 0 {
			if int(event.Fd) == s.fd.asInt() {
				return nil, os.ErrClosed
			}
			events[i].server = s
			events[i].kind = EventClose
			events[i].conn = fd(event.Fd).asConn(nil)
			i += 1
			continue
		}
		if event.Events&(EpollIN|EpollET) != 0 {
			if int(event.Fd) == s.fd.asInt() {
				conn, err := s.accept()
				if err != nil {
					continue
				}
				events[i].server = s
				events[i].kind = EventConn
				events[i].conn = conn
				i += 1
				continue
			}
			events[i].server = s
			events[i].kind = EventData
			events[i].conn = fd(event.Fd).asConn(nil)
			i += 1

		}
	}
	return events[:i], nil
}
func (s *Server) Close() error {
	s.epoll.close()
	return s.fd.close()
}
func New(addr syscall.Sockaddr) (server *Server, err error) {
	var sock int
	switch addr.(type) {
	case *syscall.SockaddrInet6:
		sock, err = syscall.Socket(syscall.AF_INET6, syscall.SOCK_STREAM, 0)
		if err != nil {
			return nil, err
		}
	default:
		sock, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		if err != nil {
			return nil, err
		}
	}
	server.fd = fd(sock)
	err = syscall.SetNonblock(sock, true)
	if err != nil {
		server.Close()
		return nil, err
	}
	err = syscall.Bind(sock, addr)
	if err != nil {
		server.Close()
		return nil, err
	}
	err = syscall.Listen(sock, 0)
	if err != nil {
		server.Close()
		return nil, err
	}
	epoll, err := syscall.EpollCreate1(0)
	if err != nil {
		server.Close()
		return nil, err
	}
	server.epoll = fd(epoll)
	event := &syscall.EpollEvent{
		Events: EpollIN | EpollHUP | EpollERR | EpollRDHUP,
		Fd:     int32(sock),
	}
	err = syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, sock, event)
	if err != nil {
		server.Close()
		return nil, err
	}
	return server, nil
}
