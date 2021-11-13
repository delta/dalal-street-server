package httpapi

import (
	"errors"
	"net/http"

	"github.com/delta/dalal-street-server/models"
)

var (
	ErrorInvalidParamter = errors.New("invalid parameters passed")
)

// HandleVerification is the view for the /verify route
/*
	TODO
	render html, with redirection to app and website
	after successfull verification
*/
func handleVerification(w http.ResponseWriter, r *http.Request) {
	verificationKey := r.URL.Query().Get("key")


	if verificationKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(ErrorInvalidParamter.Error()))
		return
	}

	if err := models.VerifyAccount(verificationKey); err != nil{
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully verified account!"))
}
