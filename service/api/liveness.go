package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) liveness(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// HTTP 204 No Content - идеальный ответ, так как он не требует тела ответа.
	w.WriteHeader(http.StatusNoContent)
}
