package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getMyConversations(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Ensure authenticated
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 2. Read optional "username" filter
	username := r.URL.Query().Get("username")
	var filterPtr *string

	if username != "" {
		// validate
		if len(username) < 3 || len(username) > 32 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "invalid username filter length",
			})
			return
		}
		if ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, username); !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "invalid username filter format",
			})
			return
		}

		filterPtr = &username
	}

	// 3. Call DB
	convs, err := rt.db.GetMyConversations(userID, filterPtr)
	if err != nil {
		ctx.Logger.Error("GetMyConversations failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 4. Send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(convs); err != nil {
		ctx.Logger.Error("encode error: ", err)
	}
}
