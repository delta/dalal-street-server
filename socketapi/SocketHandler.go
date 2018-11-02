package socketapi

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"

	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/socketapi/repl"
	"github.com/delta/dalal-street-server/utils"
)

var socketApiLogger *logrus.Entry
var upgrader websocket.Upgrader

// Init configures the socketapi package
func Init(config *utils.Config) {
	socketApiLogger = utils.Logger.WithFields(logrus.Fields{
		"module": "socketapi/SocketHandler",
	})

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if config.Stage == "test" || config.Stage == "dev" || config.Stage == "docker" {
				return true
			} else if r.Header.Get("Origin") == "https://dwst.github.io" {
				return true
			}
			return false
		},
	}
	//actions.InitActions()
	//datastreams.InitDataStreams()
	repl.InitREPL()
}

// loadSession loads a given session from the http request using the sid cookie
func loadSession(r *http.Request) (session.Session, error) {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "loadSession",
	})

	sidCookie, _ := r.Cookie("sid")
	if sidCookie != nil {
		l.Debugf("Found sid cookie")
		s, err := session.Load(sidCookie.Value)

		if err != nil {
			l.Errorf("Error loading session data: '%s'", err)
			return nil, err
		}

		l.Debugf("Loaded session")
		return s, nil
	}

	s, err := session.New()
	if err != nil {
		l.Errorf("Error starting new session: '%s'", err)
		return nil, err
	}
	l.Debugf("Created new session")

	return s, nil
}

// Handle handles an HTTP request meant for a websocket connection
func Handle(w http.ResponseWriter, r *http.Request) {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method": "Handle",
	})

	l.Infof("Connection from %+v", r.RemoteAddr)

	sess, err := loadSession(r)
	if err != nil {
		l.Errorf("Could not load or create session. Replying with 500. '%+v'", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sess.Set("IP", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, http.Header{"Set-Cookie": {"sid=" + sess.GetID() + "; HttpOnly"}})

	if err != nil {
		l.Errorf("Could not upgrade connection: '%s'", err)
		return
	}
	l.Debugf("Upgraded to websocket protocol")

	c := NewClient(make(chan struct{}), make(chan interface{}, 20), conn, sess)

	go c.WritePump()
	c.ReadPump()
}
