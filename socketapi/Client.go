package socketapi

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"

	"github.com/thakkarparth007/dalal-street-server/session"
	// socketapi_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build"
	"github.com/thakkarparth007/dalal-street-server/socketapi/repl"
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
	send chan interface{}
	done chan struct{}
	id   uuid.UUID
}

type Client interface {
	WritePump()
	ReadPump()
	Send() chan interface{}
	Done() <-chan struct{}
	GetSession() session.Session
	GetUUID() string
}

func NewClient(done chan struct{}, send chan interface{}, conn *websocket.Conn, sess session.Session) Client {
	uuid, _ := uuid.NewV4()
	return &client{
		conn: conn,
		sess: sess,
		send: send,
		done: done,
		id:   uuid,
	}
}

func (c *client) Done() <-chan struct{} {
	return c.done
}

func (c *client) Send() chan interface{} {
	return c.send
}

func (c *client) GetSession() session.Session {
	return c.sess
}

func (c *client) GetUUID() string {
	return c.id.String()
}

func (c *client) ReadPump() {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "client.ReadPump",
	})

	defer func() {
		l.Debugf("Closing connection for %+v", c.sess)
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
		msgType, bytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				l.Errorf("Error in receving from websocket: '%v'.\nClient: %+v", err, c)
			}
			break
		}

		// Admin commands come as text messages
		if msgType == websocket.TextMessage {
			if string(bytes) == "ping" {
				c.send <- "pong"
			} else {
				c.send <- repl.Handle(c.done, c.sess, string(bytes))
			}
			continue
		}

		c.conn.Close()

		/*dm := &socketapi_proto.DalalMessage{}
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
		}*/
	}
}

func (c *client) WritePump() {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "client.WritePump",
	})

	pingTicker := time.NewTicker(pingPeriod)

	defer func() {
		l.Debugf("Closing connection for %+v", c.sess)
		pingTicker.Stop()
		c.conn.Close()
		close(c.done)
	}()

	/*var sendBinary = func(msg interface{}) error {
		w, err := c.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			l.Errorf("Unable to get NextWriter. Stopping. '%+v'", err)
			return err
		}
		w.Write(msg.([]byte))
		n := len(c.send)
		for i := 0; i < n; i++ {
			w.Write((<-c.send).([]byte))
		}
		if err := w.Close(); err != nil {
			l.Errorf("Error closing the Writer. Stopping. '%+v'", err)
			return err
		}
		return nil
	}*/

	var sendText = func(msg interface{}) error {
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			l.Errorf("Unable to get NextWriter. Stopping. '%+v'", err)
			return err
		}
		w.Write([]byte(msg.(string)))
		n := len(c.send)
		for i := 0; i < n; i++ {
			w.Write([]byte((<-c.send).(string))) // interface->string->[]byte
		}
		if err := w.Close(); err != nil {
			l.Errorf("Error closing the Writer. Stopping. '%+v'", err)
			return err
		}
		return nil
	}

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			switch t := msg.(type) {
			case string:
				if err := sendText(msg); err != nil {
					return
				}
			/*case []byte:
			if err := sendBinary(msg); err != nil {
				return
			}*/
			default:
				//l.Errorf("Message should be either string or []byte. Given %t", t)
				l.Errorf("Message should be a string. Given %t", t)
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
