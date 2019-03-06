package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
)

// AddToGeneralLog adds entries to the GeneralLogs table
func AddToGeneralLog(id uint32, k string, v string) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "AddToGeneralLog",
		"k":      k,
		"v":      v,
	})

	db := utils.GetDB()

	db = db.Exec("INSERT INTO GeneralLogs VALUES (?,?,?) ON DUPLICATE KEY UPDATE `id`=?, `key`=?, `value`=?",
		id, k, v, id, k, v)

	if db.Error != nil {
		l.Errorf("Error in setting-value query: '%s'", db.Error)
		return db.Error
	}

	l.Debugf("Set key in database")
	return nil
}
