package main

import (
	"fmt"
	"net/http"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/socketapi"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func main() {
	utils.InitConfiguration("config.json")

	if utils.Configuration.Stage != "prod" {
		fmt.Println("WARNING: Server not running in prod stage.")
	}

	utils.InitLogger()

	models.InitModels()
	session.InitSession()
	socketapi.InitSocketApi()
	models.InitMatchingEngine()

	go StartREPL()
	go models.UpdateLeaderboardTicker()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", socketapi.Handle)

	port := fmt.Sprintf(":%d", utils.Configuration.HttpPort)
	utils.Logger.Fatal(http.ListenAndServe(port, nil))
}

type replCmdFn func()

var replCmds = map[string]replCmdFn{
	"sendnotif": func() {
		var userId uint32
		var text string

		fmt.Println("Enter userId and notification text:")
		fmt.Scanf("%d %q\n", &userId, &text)

		u, err := models.GetUserCopy(userId)
		if err != nil {
			fmt.Printf("Error: No user with id %d\n", userId)
			return
		}
		fmt.Printf("Are you sure you want to send '%s' to %s (userid: %d)? [Y/N]\n", text, u.Name, u.Id)

		c := 'N'
		fmt.Scanf("%c\n", &c)
		if c == 'Y' {
			err := models.SendNotification(userId, text)
			if err != nil {
				fmt.Printf("Sent\n")
				return
			}
			fmt.Println(err)
		} else {
			fmt.Printf("Not sending\n")
		}
	},
	"add_stocks_to_exchange": func() {
		var stockId uint32
		var newStocks uint32

		fmt.Println("Enter stock id and number of new stocks:")
		fmt.Scanf("%d %d\n", &stockId, &newStocks)

		s, err := models.GetStockCopy(stockId)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Are you sure you want to add %d new stocks to exchange for %s? [Y/N]", newStocks, s.FullName)

		c := 'N'
		fmt.Scanf("%c\n", &c)
		if c == 'Y' {
			err := models.AddStocksToExchange(stockId, newStocks)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Done")
		} else {
			fmt.Println("Not doing")
		}
	},
	"update_stock_price": func() {
		var stockId uint32
		var newPrice uint32

		fmt.Println("Enter stockId and new price:")
		fmt.Scanf("%d %d\n", &stockId, &newPrice)

		s, err := models.GetStockCopy(stockId)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Are you sure you want to update %s's price to %d? [Y/N]", s.FullName, newPrice)

		c := 'N'
		fmt.Scanf("%c\n", &c)
		if c == 'Y' {
			err := models.UpdateStockPrice(stockId, newPrice)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Done")
		} else {
			fmt.Println("Not doing")
		}
	},
}

func StartREPL() {
	var cmds []string
	for cmd := range replCmds {
		cmds = append(cmds, cmd)
	}

	for {
		cmd := ""
		fmt.Printf("> ")
		fmt.Scanf("%s", &cmd)
		if fn, ok := replCmds[cmd]; ok {
			fn()
			continue
		}
		fmt.Printf("Unsupported command '%s'. Only %+v are supported\n", cmd, cmds)
	}
}
