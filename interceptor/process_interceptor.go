package interceptor

import (
	"net/http"
)

//HandlerInterceptor for /external-interceptor
func HandlerInterceptor(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		rw.WriteHeader(200)
		rw.Write([]byte("ok"))
	}
}
