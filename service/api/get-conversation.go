package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getConversation(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Authentication
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 2. Extract conversationId from path
	conversationID := ps.ByName("id")

	// Validate ID length
	if len(conversationID) < 10 || len(conversationID) > 30 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid conversation ID length",
		})
		return
	}

	// Validate pattern ^[a-zA-Z0-9]+$
	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, conversationID); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid conversation ID format",
		})
		return
	}

	// 3. Query DB
	msgs, err := rt.db.GetConversation(userID, conversationID)
	if err != nil {
		if err.Error() == "conversation not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "conversation not found",
			})
			return
		}

		// some DB/internal error
		ctx.Logger.Error("GetConversation failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 4. Success → return 200 + array of messages
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		ctx.Logger.Error("encode error: ", err)
	}
}
