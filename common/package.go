package common

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

//Pkg is the unit in which messages are transferred
//between the clients and the server.
type Pkg struct {
	Type    string
	Content []byte
}

type PkgConn struct {
	conn *net.TCPConn
	dec  *json.Decoder
	enc  *json.Encoder
	//closefn  func(*PkgConn)
	CanClose *sync.Cond
	IsClosed bool
}

func NewPkgConn(conn *net.TCPConn /*, closer func(*PkgConn)*/) *PkgConn {
	conn.SetReadDeadline(time.Now().Add(time.Minute))
	return &PkgConn{
		conn:     conn,
		dec:      json.NewDecoder(conn),
		enc:      json.NewEncoder(conn),
		IsClosed: false,
		//closefn:  closer,
	}
}

//MAYBE: change to a binary protocol by sending
//[type(1Byte)[,size(8Byte),msg(sizeByte)]] where
//type is one of {<close>, <join>, <msg>, <ping>}
func (c *PkgConn) SendPkg(p *Pkg) {
	if c.IsClosed {
		return
	}

	c.conn.SetWriteDeadline(time.Now().Add(time.Minute))

	err := c.enc.Encode(p)
	if err != nil {
		c.Close()
	}

}

func (c *PkgConn) RecvPkg() (p *Pkg) {
	if c.IsClosed {
		return nil
	}
	err := c.dec.Decode(p)
	if err != nil {
		c.Close()
	}
	return
}

func (c *PkgConn) Close() {
	if !c.IsClosed {
		c.IsClosed = true
		//c.closefn(c)
		c.conn.Close()
		c.CanClose.Broadcast()
	}
}