package httpapi

import (
	"errors"
	"net/http"

	"github.com/delta/dalal-street-server/models"
)

var (
	InvalidParamterError = errors.New("Invalid parameters passed")
)

// HandleVerification is the view for the /verify route
func HandleVerification(req *http.Request) error {
	verificationKey := req.URL.Query().Get("key")

	if verificationKey == "" {
		return InvalidParamterError
	}

	err := models.VerifyAccount(verificationKey)
	return err
}
