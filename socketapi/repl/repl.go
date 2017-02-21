package repl

import (
	"fmt"
	"sync"

	"github.com/thakkarparth007/dalal-street-server/models"
)

var validCmds []string

type cmdSession struct {
	in  chan string
	out chan string
}

var cmdSessionsMutex = sync.Mutex{}
var cmdSessions = make(map[string]cmdSession)

type replCmdFn func(sess cmdSession)

func (s cmdSession) read(format string, args ...interface{}) {
	if _, err := fmt.Sscanf(<-s.in, format, args...); err != nil {
		s.error("Invalid input")
	}
}

func (s cmdSession) print(format string, args ...interface{}) {
	s.out <- fmt.Sprintf(format, args...)
}

func (s cmdSession) error(strOrErr interface{}, args ...interface{}) {
	format := ""
	switch strOrErr.(type) {
	case string:
		format = strOrErr.(string)
	case error:
		format = strOrErr.(error).Error()
	default:
		format = fmt.Sprintf("%+v", strOrErr)
	}
	s.print("Error: '"+format+"'", args...)
	panic(1) // Will be recovered below. Chill. Don't panic.
}

func (s cmdSession) finish(format string, args ...interface{}) {
	s.print(format, args...)
	panic(0) // Easy way to exit a function. :P
}

var replCmds = map[string]replCmdFn{
	"sendnotif": func(s cmdSession) {
		var userId uint32
		var text string

		s.print("Enter userId and notification text:")
		s.read("%d %q", &userId, &text)

		u, err := models.GetUserCopy(userId)
		if err != nil {
			s.error("No user with id %d", userId)
		}

		s.print("Are you sure you want to send '%s' to %s (userid: %d)? [Y/N]", text, u.Name, u.Id)

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			err := models.SendNotification(userId, text)
			if err != nil {
				s.error(err)
			}
			s.finish("Sent")
		}
		s.finish("Not sending")
	},
	"add_stocks_to_exchange": func(s cmdSession) {
		var stockId uint32
		var newStocks uint32

		s.print("Enter stock id and number of new stocks:")
		s.read("%d %d\n", &stockId, &newStocks)

		stock, err := models.GetStockCopy(stockId)
		if err != nil {
			s.error(err)
		}

		s.print("Are you sure you want to add %d new stocks to exchange for %s? [Y/N]", newStocks, stock.FullName)

		c := 'N'
		s.read("%c\n", &c)
		if c == 'Y' {
			err := models.AddStocksToExchange(stockId, newStocks)
			if err != nil {
				s.error(err)
			}
			s.finish("Done")
		}
		s.finish("Not doing")
	},
	"update_stock_price": func(s cmdSession) {
		var stockId uint32
		var newPrice uint32

		s.print("Enter stockId and new price:")
		s.read("%d %d\n", &stockId, &newPrice)

		stock, err := models.GetStockCopy(stockId)
		if err != nil {
			s.error(err)
		}

		s.print("Are you sure you want to update %s's price to %d? [Y/N]", stock.FullName, newPrice)

		c := 'N'
		s.read("%c\n", &c)
		if c == 'Y' {
			err := models.UpdateStockPrice(stockId, newPrice)
			if err != nil {
				s.error(err)
			}
			s.finish("Done")
		}
		s.finish("Not doing")
	},
}

func init() {
	for cmd := range replCmds {
		validCmds = append(validCmds, cmd)
	}
}

func Handle(done <-chan struct{}, sid string, cmd string) string {
	cmdSessionsMutex.Lock()
	defer cmdSessionsMutex.Unlock()

	if session, ok := cmdSessions[sid]; !ok {
		if _, isValid := replCmds[cmd]; !isValid {
			return fmt.Sprintf("Invalid command '%s'. Valid commands are: %+v ", cmd, validCmds)
		}

		cmdSessions[sid] = cmdSession{
			in:  make(chan string),
			out: make(chan string, 1), // so that the command doesn't hang if `done` closes before its output is read
		}
		session = cmdSessions[sid]

		// launch the command
		go func() {
			defer func() {
				recover() // to be ignored. Both panics above are exit-hacks
				cmdSessionsMutex.Lock()
				delete(cmdSessions, sid)
				cmdSessionsMutex.Unlock()
			}()
			replCmds[cmd](session)
		}()

		// Start the cleanup go routine. Its only job is to remove the session when either the input or the output is done.
		go func() {
			// if the client closed connection, there's no input. Inform the command that there's no more input
			<-done
			cmdSessionsMutex.Lock()
			close(cmdSessions[sid].in)
			cmdSessionsMutex.Unlock()
		}()

		return <-session.out
	}

	sess := cmdSessions[sid]
	select {
	case <-done:
		// do nothing. Client has closed. Don't send the input to the command. Let the cleanup listener close the session
		return ""
	default:
		// the client hasn't closed yet. Send the input to the command.
		sess.in <- cmd
		return <-sess.out // safe to return command's output here since the input is sent.
	}
}
