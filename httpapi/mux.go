package httpapi

import (
	"net/http"
)

var HttpMux *http.ServeMux

func Init() {
	HttpMux = http.NewServeMux()

	// verification route
	HttpMux.HandleFunc("/verify",handleVerification)

	//serve public dir
	HttpMux.Handle("/static/",http.StripPrefix("/static/",http.FileServer(http.Dir("./public"))))
}
