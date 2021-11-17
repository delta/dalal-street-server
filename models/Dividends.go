package models

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

//userDetails stores users who own stocks of a given stockid
type userDetails struct {
	UserID        uint32
	StockQuantity uint64
}

//PerformDividendsTransaction finds the users who own the stocks of this Company,
//                            performs dividendTransaction for each of these users,
//                            updates cash of each of these Users,
//                            sends the DividendTransactions through datastreams and a notification.
func PerformDividendsTransaction(stockID uint32, dividendAmount uint64) error {
	var l = logger.WithFields(logrus.Fields{
		"method":               "PerformDividendTransaction",
		"param_stockId":        stockID,
		"param_dividendAmount": dividendAmount,
	})

	var transactions []*Transaction

	dividendsMap := make(map[uint32]uint64)

	l.Infof("PerformDividendTransaction requested")

	db := getDB()

	maxStockID, err := GetmaxStockID()

	l.Infof("Max stock id : %v", maxStockID)

	if err != nil {
		l.Errorf("Failure to get max stock id due to : %v+", err)
		return err
	}

	if stockID < 0 || stockID > maxStockID {
		l.Errorf("Failure with stock id : %v", stockID)
		return InvalidStockIdError{}
	}

	/* Committing to database */
	tx := db.Begin()

	errorHelper := func(format string, args ...interface{}) error {
		l.Errorf(format, args...)
		tx.Rollback()
		return fmt.Errorf(format, args...)
	}

	var benefittingUsers []*userDetails

	db.Raw("SELECT userId As user_id ,(SUM(stockQuantity)+SUM(reservedStockQuantity)) AS stock_quantity FROM Transactions WHERE stockId = ? GROUP BY userId HAVING stock_quantity > 0 ", stockID).Scan(&benefittingUsers)

	l.Infof("Successfully fetched users who own stocks of stockID : %v", stockID)

	for _, user := range benefittingUsers {

		dividendTotal := dividendAmount * uint64(user.StockQuantity)
		l.Infof(" The user id is %v and stock quantity is : %v", user.UserID, user.StockQuantity)

		transaction := GetTransactionRef(user.UserID, stockID, DividendTransaction, 0, 0, uint64(dividendAmount), 0, int64(dividendTotal))

		if err := tx.Save(transaction).Error; err != nil {
			errRevert := RevertToOldState(dividendsMap)
			if errRevert != nil {
				return errRevert
			}
			return errorHelper("Error creating the transaction. Rolling back. Error: %+v", err)
		}

		l.Debugf("Added transaction to Transactions table")

		l.Debugf("Acquiring exclusive write on user")
		ch, currentUser, err := getUserExclusively(user.UserID)
		if err != nil {
			l.Errorf("Errored : %+v ", err)
			errRevert := RevertToOldState(dividendsMap)
			if errRevert != nil {
				return errRevert
			}
			return errorHelper("Error acquring exclusive write on user. Rolling back. Error: %+v", err)
		}
		l.Debugf("Acquired")
		defer func(ch chan struct{}) {
			close(ch)
			l.Debugf("Released exclusive write on user")
		}(ch)

		// A lock on user and stock has been acquired.
		// Safe to make changes to this user and this stock
		dividendsMap[currentUser.Id] = dividendTotal
		currentUser.Cash = uint64(int64(currentUser.Cash) + int64(dividendTotal))

		if err := tx.Save(currentUser).Error; err != nil {
			errRevert := RevertToOldState(dividendsMap)
			if errRevert != nil {
				return errRevert
			}
			return errorHelper("Error updating user's cash. Rolling back. Error: %+v", err)
		}

		transactions = append(transactions, transaction)
		l.Infof("Updated user's cash. New balance: %d", currentUser.Cash)
	}

	if err := tx.Commit().Error; err != nil {
		errRevert := RevertToOldState(dividendsMap)
		if errRevert != nil {
			return errRevert
		}
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Debugf("Committed transaction. Success.")

	errNotif := SendDividendsTransactionsAndNotifications(transactions, benefittingUsers, stockID)
	if errNotif != nil {
		return errNotif
	}

	return nil

}

//RevertToOldState restores oldCash for all benefittingUsers in User Map
func RevertToOldState(dividendsMap map[uint32]uint64) error {
	var l = logger.WithFields(logrus.Fields{
		"method":             "RevertToOldState",
		"param_dividendsMap": fmt.Sprintf("%+v", dividendsMap),
	})

	for userID, dividend := range dividendsMap {
		ch, currentUser, err := getUserExclusively(userID)
		if err != nil {
			l.Errorf("Errored : %+v ", err)
			return err
		}
		l.Debugf("Acquired")
		defer func(ch chan struct{}) {
			close(ch)
			l.Debugf("Released exclusive write on user")
		}(ch)
		currentUser.Cash = currentUser.Cash - dividend
	}
	return nil
}

// SendDividendsTransactionsAndNotifications sends Transactions through Transactions stream and a Notification about Dividends to all users.
func SendDividendsTransactionsAndNotifications(transactions []*Transaction, benefittingUsers []*userDetails, stockID uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":                 "SendTransactionsAndNotifications",
		"param_stockId":          stockID,
		"param_transactions":     fmt.Sprintf("%+v", transactions),
		"param_benefittingUsers": fmt.Sprintf("%+v", benefittingUsers),
	})
	for _, tr := range transactions {
		go func(transaction Transaction) {
			transactionsStream := datastreamsManager.GetTransactionsStream()
			transactionsStream.SendTransaction(transaction.ToProto())
			l.Infof("Sent transactions through the datastreams")
		}(*tr)
	}

	stockDetails, err := GetCompanyDetails(stockID)
	if err != nil {
		l.Errorf("Getting Company name failed due to %+v: ", err)
		return err
	}

	SendPushNotification(0, PushNotification{
		Title:   "Message from Dalal Street! A Company just sent our dividends.",
		Message: fmt.Sprintf("Company %+v has sent out dividends. Visit the site to claim your reward.", stockDetails.FullName),
		LogoUrl: fmt.Sprintf("%v/static/dalalfavicon.png", config.BackendUrl),
	})
	SendNotification(0, fmt.Sprintf("Company %+v has sent out dividends.", stockDetails.FullName), true)
	return nil
}

//GetmaxStockID returns the maximum StockID from the stocks.
func GetmaxStockID() (uint32, error) {
	db := getDB()
	sql := "SELECT MAX(id) as max_stockId FROM Stocks"
	rows, err := db.Raw(sql).Rows()
	defer rows.Close()
	if err != nil {
		return 0, err
	} else {
		var maxStockID uint32
		for rows.Next() {
			rows.Scan(&maxStockID)
		}
		return maxStockID, nil
	}
}
