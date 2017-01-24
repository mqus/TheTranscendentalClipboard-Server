package srv

import (
	"errors"
	"log"
	"net"
	"sync"

	"github.com/mqus/TheTranscendentalClipboard-Server/common"
)

// A Room knows all the clients which share the same Clipboard.
type Room struct {
	name    string
	clients map[int]*Client
	maxid   int
	mutex   sync.RWMutex
}

// A Client knows his connection(including all the neccessary en/decoders) and his position in the parent room.
// the position is important to not send the same clipboard message back (and inducing some ugly race conditions)
type Client struct {
	conn *common.PkgConn
	id   int
	room *Room
}

var (
	// ErrConnClosed is thrown when the package size cannot be read/written or the package cannot be read/written entirely.
	ErrConnClosed = errors.New("client lost/closed the connection")
	rooms         = make(map[string]*Room)
	roomsMutex    sync.RWMutex
)

// AddClient adds a new Client to the system by encoding the first message with the room name and then
// assigning the Client to the room.
func AddClient(conn *net.TCPConn) {
	tc := common.NewPkgConn(conn)

	pkg := tc.RecvPkg()
	if tc.IsClosed {
		log.Println("Client was closed before the room was assigned... suspicious...")
		return
	}
	if pkg.Type != "hello" {
		tc.Close()
		log.Println("Client sent the wrong first message:", pkg)
		return
	}
	roomname := string(pkg.Content)
	room := getRoom(roomname)
	c := Client{
		conn: tc,
	}

	//add Client to Room and the room to the client
	room.mutex.Lock()
	c.id = room.maxid
	room.maxid++

	room.clients[c.id] = &c
	c.room = room

	room.mutex.Unlock()
	go waitForClosing(&c)

	c.recvLoop()
}

func getRoom(name string) (room *Room) {
	roomsMutex.RLock()
	room = rooms[name]
	roomsMutex.RUnlock()

	//if room is not there already, add it
	if room == nil {
		room = &Room{
			name:    name,
			clients: make(map[int]*Client),
			maxid:   0,
		}

		roomsMutex.Lock()
		rooms[name] = room
		roomsMutex.Unlock()

	}
	return room
}

func (c *Client) recvLoop() {
	for {
		pkg := c.conn.RecvPkg()
		if c.conn.IsClosed {
			return
		}
		if pkg.Type != "pkg" {
			c.room.mutex.RLock()
			clients := c.room.clients
			c.room.mutex.RUnlock()
			for id, client := range clients {
				if id != c.id {
					client.conn.SendPkg(pkg)
				}
			}
		}
	}
}

func waitForClosing(c *Client) {
	//Wait till PkgConn has closed the connection and then close from this end.
	c.conn.CanClose.Wait()
	if c.room != nil {
		//delete client from room
		c.room.mutex.Lock()
		delete(c.room.clients, c.id)
		c.room.mutex.Unlock()

		//MAYBE: implement some kind of GC
		/* IS NOT SAFE TO DO, THEREFORE COMMENTED OUT
		c.room.mutex.RLock()

		//if the room is now empty, delete the room.
		if len(c.room.clients) == 0 and c.room.{
			roomsMutex.Lock()
			delete(rooms, c.room.name)
			////////////////22222
			roomsMutex.Lock()
		}
		c.room.mutex.RUnlock()*/
	}
}