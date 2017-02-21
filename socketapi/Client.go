package socketapi

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"

	"github.com/thakkarparth007/dalal-street-server/session"
	socketapi_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build"
)

const (
	maxMessageSize = 512
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

type client struct {
	conn *websocket.Conn
	sess session.Session
	send chan []byte
	done chan struct{}
	id   uuid.UUID
}

type Client interface {
	WritePump()
	ReadPump()
	Send() chan []byte
	Done() <-chan struct{}
	GetSession() session.Session
}

func NewClient(done chan struct{}, send chan []byte, conn *websocket.Conn, sess session.Session) Client {
	return &client{
		conn: conn,
		sess: sess,
		send: send,
		done: done,
		id:   uuid.NewV4(),
	}
}

func (c *client) Done() <-chan struct{} {
	return c.done
}

func (c *client) Send() chan []byte {
	return c.send
}

func (c *client) GetSession() session.Session {
	return c.sess
}

func (c *client) ReadPump() {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "client.ReadPump",
	})

	defer func() {
		c.conn.Close()
		//close(c.done)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				l.Errorf("Error in receving from websocket: '%v'.\nClient: %+v", err, c)
			}
			break
		}

		dm := &socketapi_proto.DalalMessage{}
		err = proto.Unmarshal(bytes, dm)
		if err != nil {
			l.Errorf("Error in unmarshaling the message. Client: '%+v'", c)
			break
		}

		switch dm.MessageType.(type) {
		case *socketapi_proto.DalalMessage_RequestWrapper:
			go handleRequest(c, dm.GetRequestWrapper())
		default:
			l.Errorf("Unmarshaled message has unexpected MessageType: %+v", dm)
			return
		}
	}
}

func (c *client) WritePump() {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "client.WritePump",
	})

	pingTicker := time.NewTicker(pingPeriod)

	defer func() {
		pingTicker.Stop()
		c.conn.Close()
		close(c.done)
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				l.Errorf("Unable to get NextWriter. Stopping. '%+v'", err)
				return
			}
			w.Write(msg)
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				l.Errorf("Error closing the Writer. Stopping. '%+v'", err)
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				l.Errorf("Error sending ping message. Stopping. '%v'", err)
				return
			}
		}
	}
}
