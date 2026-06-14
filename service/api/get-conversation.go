package api

import (
	"encoding/json"
	"net/http"

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

	// Validate ID length (Already fixed to 64!)
	if len(conversationID) < 3 || len(conversationID) > 64 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid conversation ID length",
		})
		return
	}

	// Validate pattern using the package-level variable from forward-message.go
	if !fwdIDRegex.MatchString(conversationID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid conversation ID format",
		})
		return
	}

	// 3. Query DB

	// --- NEW LINE: Mark messages as read when opening chat ---
	// We explicitly ignore the error here (errcheck compliant)
	_ = rt.db.SetConversationRead(userID, conversationID)
	// --------------------------------------------------------

	msgs, err := rt.db.GetConversation(userID, conversationID)
	if err != nil {
		if err.Error() == "conversation not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "conversation not found",
			})
			return
		}
		if err.Error() == "forbidden" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "not a member of this group",
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
