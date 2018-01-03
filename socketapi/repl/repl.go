package repl

import (
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry
var validCmds []string

type cmdSession struct {
	in  chan string
	out chan string
}

var cmdSessionsMutex = sync.Mutex{}
var cmdSessions = make(map[string]cmdSession)

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

type replCmdFn func(userSess session.Session, sess cmdSession)

var replCmds = map[string]replCmdFn{
	"login": func(userSess session.Session, s cmdSession) {
		var username string
		var password string

		s.print("Are you a control freak?")
		s.read("yes")

		s.print("Who's the boss?")
		s.read("Parth")

		s.print("Good. Enter your username: ")
		s.read("%q", &username)

		s.print("Password?")
		s.read("%q", &password)

		if isAdmin, err := models.IsAdmin(username, password); err != nil {
			models.AdminLog("", err.Error())
			s.error("Error")
		} else if !isAdmin {
			s.finish(" :) -> :(")
		}

		models.AdminLog(username, "Logged in")

		userSess.Set("repl_Username", username)
		userSess.Set("repl_IsAdmin", "true")
		s.finish("Welcome.")
	},
	"logout": func(userSess session.Session, s cmdSession) {
		aun, _ := userSess.Get("repl_Username")
		userSess.Delete("repl_IsAdmin")
		userSess.Delete("repl_Username")
		models.AdminLog(aun, "Logged out")
		s.finish("Goodbye.")
	},
	"open_market": func(userSess session.Session, s cmdSession) {
		aun, _ := userSess.Get("repl_Username")

		s.print("Are you sure you want to open the market?")

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			models.OpenMarket()
			models.AdminLog(aun, "Opened market")
			s.finish("Done")
		}

		s.finish("Not doing")
	},
	"close_market": func(userSess session.Session, s cmdSession) {
		aun, _ := userSess.Get("repl_Username")

		s.print("Are you sure you want to close the market?")

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			models.CloseMarket()
			models.AdminLog(aun, "Closed market")
			s.finish("Done")
		}

		s.finish("Not doing")
	},
	"load_stocks": func(userSess session.Session, s cmdSession) {
		aun, _ := userSess.Get("repl_Username")

		s.print("Are you sure you want to reload the stocks?")

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			models.LoadStocks()
			models.AdminLog(aun, "Reloading the stocks")
			s.finish("Done")
		}

		s.finish("Not doing")
	},
	"sendnotif": func(userSess session.Session, s cmdSession) {
		var userId uint32
		var isGlobal bool
		var text string

		aun, _ := userSess.Get("repl_Username")

		s.print("Enter userId and notification text:")
		s.read("%d %q", &userId, &text)

		if userId == 0 {
			isGlobal = true
			s.print("Are you sure you want to send '%s' to ALL users?", text)
		} else {
			u, err := models.GetUserCopy(userId)
			if err != nil {
				s.error("No user with id %d", userId)
			}

			s.print("Are you sure you want to send '%s' to %s (userid: %d)? [Y/N]", text, u.Name, u.Id)
		}

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			err := models.SendNotification(userId, text, isGlobal)
			if err != nil {
				models.AdminLog(aun, fmt.Sprintf("Sending '%s' to %d failed. Error: %+v", text, userId, err))
				s.error(err)
			}
			models.AdminLog(aun, fmt.Sprintf("Send '%s' to %d", text, userId))
			s.finish("Sent")
		}
		s.finish("Not sending")
	},
	"add_stocks_to_exchange": func(userSess session.Session, s cmdSession) {
		var stockId uint32
		var newStocks uint32

		aun, _ := userSess.Get("repl_Username")

		s.print("Enter stock id and number of new stocks:")
		s.read("%d %d\n", &stockId, &newStocks)

		stock, err := models.GetStockCopy(stockId)
		if err != nil {
			s.error(err)
		}

		s.print("Are you sure you want to add %d new stocks to exchange for %s? [Y/N]", newStocks, stock.FullName)

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			err := models.AddStocksToExchange(stockId, newStocks)
			if err != nil {
				models.AdminLog(aun, fmt.Sprintf("Adding %d stocks to stockid %d failed due to '%+v'", newStocks, stockId, err))
				s.error(err)
			}
			models.AdminLog(aun, fmt.Sprintf("Added %d stocks to stockid %d", newStocks, stockId))
			s.finish("Done")
		}
		s.finish("Not doing")
	},
	"update_stock_price": func(userSess session.Session, s cmdSession) {
		var stockId uint32
		var newPrice uint32

		aun, _ := userSess.Get("repl_Username")

		s.print("Enter stockId and new price:")
		s.read("%d %d", &stockId, &newPrice)

		stock, err := models.GetStockCopy(stockId)
		if err != nil {
			s.error(err)
		}

		s.print("Are you sure you want to update %s's price to %d? [Y/N]", stock.FullName, newPrice)

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			err := models.UpdateStockPrice(stockId, newPrice)
			if err != nil {
				models.AdminLog(aun, fmt.Sprintf("Updating stock price of %d to Rs. %d failed due to '%+v'", stockId, newPrice, err))
				s.error(err)
			}
			models.AdminLog(aun, fmt.Sprintf("Update stock price of %d to Rs. %d", stockId, newPrice))
			s.finish("Done")
		}
		s.finish("Not doing")
	},
	"add_market_event": func(userSess session.Session, s cmdSession) {
		var stockId uint32
		var headline string
		var text string
		var isGlobal bool

		aun, _ := userSess.Get("repl_Username")

		s.print("Enter stockId and headline:")
		s.read("%d %q", &stockId, &headline)

		s.print("Enter brief text:")
		s.read("%q", &text)

		if stockId == 0 {
			isGlobal = true
		} else {
			stock, err := models.GetStockCopy(stockId)
			if err != nil {
				s.error(err)
			}

			s.print("Are you sure you want to send '%s'[%s] for '%s'? [Y/N]", headline, text, stock.FullName)
		}

		c := 'N'
		s.read("%c", &c)
		if c == 'Y' {
			err := models.AddMarketEvent(stockId, headline, text, isGlobal)
			if err != nil {
				models.AdminLog(aun, fmt.Sprintf("Adding market event '%s'[%s] for %d failed due to '%+v'", headline, text, stockId, err))
				s.error(err)
			}
			models.AdminLog(aun, fmt.Sprintf("Added market event '%s'[%s] for %d", headline, text, stockId))
			s.finish("Done")
		}
		s.finish("Not doing")
	},
}

func InitREPL() {
	logger := utils.Logger.WithFields(logrus.Fields{
		"module": "socketapi/repl",
	})
	for cmd := range replCmds {
		validCmds = append(validCmds, cmd)
	}
	logger.Info("REPL Started")
}

func Handle(done <-chan struct{}, userSess session.Session, cmd string) (ret string) {
	var l = utils.Logger.WithFields(logrus.Fields{
		"method":         "Handle",
		"param_userSess": userSess,
		"param_cmd":      cmd,
	})

	defer func() {
		if r := recover(); r != nil {
			ret = "REPL Panicked! Ignoring this to save the server from death."
			l.Errorf("Something really bad happened. Stack: %s", string(debug.Stack()))
		}
	}()

	sid := userSess.GetId()

	cmdSessionsMutex.Lock()
	defer cmdSessionsMutex.Unlock()

	if _, ok := cmdSessions[sid]; !ok {
		if _, isValid := replCmds[cmd]; !isValid {
			return fmt.Sprintf("Invalid command '%s'. Valid commands are: %+v ", cmd, validCmds)
		}

		if _, ok := userSess.Get("repl_IsAdmin"); !ok && cmd != "login" {
			return "Tata!"
		}

		cmdSessions[sid] = cmdSession{
			in:  make(chan string),
			out: make(chan string, 1), // so that the command doesn't hang if `done` closes before its output is read
		}
		cmdSess := cmdSessions[sid]

		// launch the command
		go func() {
			defer func() {
				recover() // to be ignored. Both panics above are exit-hacks
				cmdSessionsMutex.Lock()
				delete(cmdSessions, sid)
				cmdSessionsMutex.Unlock()
			}()
			replCmds[cmd](userSess, cmdSess)
		}()

		// Start the cleanup go routine. Its only job is to remove the session when either the input or the output is done.
		go func() {
			// if the client closed connection, there's no input. Inform the command that there's no more input
			<-done
			userSess.Delete("repl_IsAdmin")
			userSess.Delete("repl_Username")
			cmdSessionsMutex.Lock()
			// important, coz if the cmdSessions[sid] gets deleted, the chan is also closed.
			if _, ok := cmdSessions[sid]; ok {
				close(cmdSessions[sid].in)
			}
			cmdSessionsMutex.Unlock()
		}()

		return <-cmdSess.out
	}

	cmdSess := cmdSessions[sid]
	select {
	case <-done:
		// do nothing. Client has closed. Don't send the input to the command. Let the cleanup listener close the session
		return ""
	default:
		// the client hasn't closed yet. Send the input to the command.
		cmdSess.in <- cmd
		return <-cmdSess.out // safe to return command's output here since the input is sent.
	}
}
