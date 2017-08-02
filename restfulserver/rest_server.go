package restfulserver

import (
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rancher/pipeline/pipeline"
)

type testHandler struct {
}

//ListenAndServe do http rest serve
func ListenAndServe(pipelineContext *pipeline.PipelineContext, errChan chan bool) {
	server := NewServer(pipelineContext)
	router := http.Handler(NewRouter(server))
	router = handlers.LoggingHandler(os.Stdout, router)
	router = handlers.ProxyHeaders(router)
	if err := http.ListenAndServe(":60080", router); err != nil {
		logrus.Error(err)
		errChan <- true
	}
}

//for webhook trigger
func ListenAndServeExternal(pipelineContext *pipeline.PipelineContext, errChan chan bool) {
	server := NewServer(pipelineContext)
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError

	//webhook
	router.Methods(http.MethodPost).Path("/v1/webhook/{id}").Handler(f(schemas, server.Webhook))
	handler := http.Handler(router)
	handler = handlers.LoggingHandler(os.Stdout, handler)
	handler = handlers.ProxyHeaders(handler)

	if err := http.ListenAndServe(":60081", handler); err != nil {
		logrus.Error(err)
		errChan <- true
	}

}

func (testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}
