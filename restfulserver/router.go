package restfulserver

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/interceptor"
)

//HandleError handle error from operation
func HandleError(s *client.Schemas, t func(http.ResponseWriter, *http.Request) error) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := t(rw, req); err != nil {
			apiContext := api.GetApiContext(req)
			apiContext.Write(err)
			//apiContext.WriteErr(err)
		}
	}))
}

//NewRouter router for schema
func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError

	// for intercepter
	router.Methods("Post").Path("/external-interceptor").HandlerFunc(interceptor.HandlerInterceptor)
	// API framework routes
	//router.Methods("GET").Path("/").Handler(api.VersionsHandler(schemas, "v1"))
	//router.Methods("GET").Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	//router.Methods("GET").Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	//router.Methods("GET").Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	router.Methods("GET").Path("/v1/pipelines/{env}/pipelines").Handler(f(schemas, s.ListPipelines))
	// Volume
	// router.Methods("GET").Path("/v1/volumes").Handler(f(schemas, s.ListVolumes))
	// router.Methods("GET").Path("/v1/volumes/{id}").Handler(f(schemas, s.GetVolume))
	// router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "readat").Handler(f(schemas, s.ReadAt))
	// router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "writeat").Handler(f(schemas, s.WriteAt))

	return router
}
