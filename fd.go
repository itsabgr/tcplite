package tcplite

import "syscall"

type fd int

func (f fd) asConn(addr syscall.Sockaddr) *Conn {
	return &Conn{
		fd:   f,
		addr: addr,
	}
}

func (f fd) asInt() int {
	return int(f)
}

func (f fd) close() error {
	return syscall.Close(f.asInt())
}
func close(fd int) error {
	return syscall.Close(fd)
}
