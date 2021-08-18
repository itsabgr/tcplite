package tcplite

import "syscall"

type Conn struct {
	fd   fd
	addr syscall.Sockaddr
}

func (r *Conn) Close() error {
	return syscall.Close(r.fd.asInt())
}

//Fd returns conn fd
func (r *Conn) Fd() int {
	return r.fd.asInt()
}

//Read impl io.Reader for conn
func (r *Conn) Read(b []byte) (int, error) {
	return syscall.Read(r.fd.asInt(), b)
}

//Write impl io.Writer for conn
func (r *Conn) Write(b []byte) (int, error) {
	return syscall.Write(r.fd.asInt(), b)
}

//Addr return the conn addr
func (r *Conn) Addr() syscall.Sockaddr {
	if r.addr != nil {
		return r.addr
	}
	addr, err := syscall.Getsockname(r.fd.asInt())
	if err != nil {
		panic(err)
	}
	r.addr = addr
	return addr
}
