package httpapi

import (
	"net/http"
)

var HttpMux *http.ServeMux

func Init() {
	HttpMux = http.NewServeMux()

	// verification route
	HttpMux.HandleFunc("/verify", handleVerification)

	//serve public dir
	HttpMux.Handle("/static/", http.StripPrefix("/static/", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		resp.Header().Add("Access-Control-Allow-Origin", "*")
		resp.Header().Add("Access-Control-Allow-Methods", "*")
		resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,x-grpc-web,sessionid,x-user-agent")
		resp.Header().Add("Access-Control-Max-Age", "600")
		http.FileServer(http.Dir("./public"))
	})))
}
