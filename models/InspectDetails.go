package models

import (
	"github.com/sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
)

func (l *InspectDetails) ToProto() *models_pb.InspectDetails {
	return &models_pb.InspectDetails{
		Id:               l.UserId,
		Email:            l.Email,
		TransactionCount: l.Count,
		Position:         l.Position,
		StockSum:         l.StockSum,
	}
}

type InspectDetails struct {
	UserId   uint32
	Email    string
	Count    uint64
	Position uint32
	StockSum int64
}

type Lrank struct {
	Position uint32
}

func GetInspectUserDetails(userID uint32, transType bool, day uint32) ([]InspectDetails, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "	GetInspectUserDetails",
		"param_userID": userID,
	})

	l.Debugf("Attempting to get user details")

	var inspectUserEntries []InspectDetails

	db := getDB()
	var err error
	if transType == false {
		err = db.Raw("SELECT u.id as user_id, u.email as email,COUNT(u.id) as count, -sum(t.reservedStockQuantity) as stock_sum FROM `OrderFills` o, `Bids` b, `Users` u, `Asks` a, `Transactions` t WHERE (o.bidId = b.id AND a.userId = ? AND a.id = o.askId AND b.userID = u.id AND o.transactionId = t.id AND datediff(current_timestamp(), t.createdAt) <= ?) GROUP BY u.id ORDER BY COUNT(u.id) DESC LIMIT 10", userID, day).Scan(&inspectUserEntries).Error
	} else {
		err = db.Raw("SELECT u.id as user_id, u.email as email,COUNT(u.id) as count, -sum(t.reservedStockQuantity) as stock_sum FROM `OrderFills` o, `Bids` b, `Users` u, `Asks` a, `Transactions` t WHERE (o.bidId = b.id AND b.userId = ? AND a.id = o.askId AND a.userID = u.id AND o.transactionId = t.id AND datediff(current_timestamp(), t.createdAt) <= ?) GROUP BY u.id ORDER BY COUNT(u.id) DESC LIMIT 10", userID, day).Scan(&inspectUserEntries).Error
	}

	for i := 0; i < len(inspectUserEntries); i++ {
		var temp Lrank
		err = db.Raw("SELECT rank as position FROM Leaderboard WHERE userId = ?", inspectUserEntries[i].UserId).Scan(&temp).Error
		inspectUserEntries[i].Position = temp.Position
	}

	if err != nil {
		l.Errorf("Attempting to get user details")
		return nil, err
	}

	return inspectUserEntries, nil

}
