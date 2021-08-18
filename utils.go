package tcplite

import (
	"fmt"
	. "github.com/itsabgr/go-handy"
	"net"
	"syscall"
)

func ResolveAddr(addr string) syscall.Sockaddr {
	parsed, err := net.ResolveTCPAddr("tcp", addr)
	Throw(err)
	switch len(parsed.IP) {
	case 4:
		return &syscall.SockaddrInet4{
			Port: parsed.Port,
			Addr: [4]byte{parsed.IP[0], parsed.IP[1], parsed.IP[2], parsed.IP[3]},
		}
	case 16:
		return &syscall.SockaddrInet6{
			Port: parsed.Port,
			Addr: [16]byte{
				parsed.IP[0], parsed.IP[1], parsed.IP[2], parsed.IP[3],
				parsed.IP[4], parsed.IP[5], parsed.IP[6], parsed.IP[7],
				parsed.IP[8], parsed.IP[9], parsed.IP[10], parsed.IP[11],
				parsed.IP[12], parsed.IP[13], parsed.IP[14], parsed.IP[15],
			},
		}
	}
	panic(fmt.Errorf("tcplite: unsupported addr %q", parsed))

}
