package restfulserver

import (
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
)

type testHandler struct {
}

//ListenAndServe do http rest serve
func ListenAndServe(errChan chan bool) {
	server := NewServer()
	router := http.Handler(NewRouter(server))
	router = handlers.LoggingHandler(os.Stdout, router)
	router = handlers.ProxyHeaders(router)
	if err := http.ListenAndServe(":60080", router); err != nil {
		logrus.Error(err)
		errChan <- true
	}
}

func (testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}
