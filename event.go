package fasttcp

import (
	"fmt"
	"syscall"
)

type EventKind = int

const (
	EventConn EventKind = iota + 1
	EventData
	EventClose
)

type Event struct {
	conn   *Conn
	server *Server
	kind   EventKind
}

func (ev *Event) Kind() EventKind {
	return ev.kind
}
func (ev *Event) Conn() *Conn {
	return ev.conn
}
func (ev *Event) Server() *Server {
	return ev.server
}

//ConnEvents are connection pulling events
var ConnEvents uint32 = EpollIN | EpollHUP | EpollERR | EpollRDHUP | EpollONESHOT | EpollET

func (ev *Event) Allow() error {
	switch ev.kind {
	case EventConn:
		event := &syscall.EpollEvent{
			Fd:     int32(ev.conn.Fd()),
			Events: ConnEvents,
		}
		return syscall.EpollCtl(ev.server.epoll.asInt(), syscall.EPOLL_CTL_ADD, ev.conn.fd.asInt(), event)
	case EventData:
		event := &syscall.EpollEvent{
			Fd:     int32(ev.conn.Fd()),
			Events: ConnEvents,
		}
		return syscall.EpollCtl(ev.server.epoll.asInt(), syscall.EPOLL_CTL_MOD, ev.conn.fd.asInt(), event)
	}
	panic(fmt.Errorf(`tcplite: can not call Allow() on "%s"`, ev.String()))
}
func (ev *Event) String() string {
	switch ev.kind {
	case EventClose:
		return "CloseEvent"
	case EventConn:
		return "ConnEvent"
	case EventData:
		return "DataEvent"
	}
	return "UnknownEvent"
}
