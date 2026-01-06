package api

import (
	"encoding/json"
	"net/http"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) deleteMessage(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Auth
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 2. Extract message ID from path
	msgID := ps.ByName("id")
	if len(msgID) < 10 || len(msgID) > 30 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid message id",
		})
		return
	}

	// 3. Call DB
	// FIXED: Arguments were swapped. Signature is DeleteMessage(messageID, userID)
	err := rt.db.DeleteMessage(msgID, userID)

	if err != nil {
		// FIXED: Use constants or variables to satisfy goconst linter
		switch err.Error() {

		case "message not found":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "message not found",
			})
			return

		case "forbidden":
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "cannot delete another user's message",
			})
			return

		default:
			ctx.Logger.Error("DB DeleteMessage error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "database error",
			})
			return
		}
	}

	// 4. Success — No Content
	w.WriteHeader(http.StatusNoContent)
}
