package api

import (
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type httpRouterHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

func (rt *_router) wrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		// Create request ID
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Build context
		var ctx = reqcontext.RequestContext{
			ReqUUID: reqUUID,
		}

		// Request-specific logger
		ctx.Logger = rt.baseLogger.WithFields(logrus.Fields{
			"reqid":     ctx.ReqUUID.String(),
			"remote-ip": r.RemoteAddr,
		})

		// ---------- Extract Bearer token ----------
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			ctx.UserID = strings.TrimPrefix(auth, "Bearer ")
		}

		// Pass context to handler
		fn(w, r, ps, ctx)
	}
}
