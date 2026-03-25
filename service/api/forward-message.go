package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

var fwdIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// Define constants to satisfy 'goconst' linter
const (
	ERR_MSG_NOT_FOUND       = "message not found"
	ERR_RECIPIENT_NOT_FOUND = "recipient not found"
)

type forwardMessageRequest struct {
	MessageID   string `json:"messageId"`
	RecipientID string `json:"recipientId"`
}

func (rt *_router) forwardMessage(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Validate token
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 2. Parse body
	var body forwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid JSON"})
		return
	}

	// 3. Validate messageId
	if len(body.MessageID) < 10 || len(body.MessageID) > 64 || !fwdIDRegex.MatchString(body.MessageID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid messageId"})
		return
	}

	// 4. Validate recipientId
	if len(body.RecipientID) < 10 || len(body.RecipientID) > 64 || !fwdIDRegex.MatchString(body.RecipientID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid recipientId"})
		return
	}

	// 5. Forward
	// FIXED: Arguments were swapped.
	// DB signature: ForwardMessage(messageID, newRecipientID, senderID)
	msg, err := rt.db.ForwardMessage(body.MessageID, body.RecipientID, userID)
	if err != nil {
		if err.Error() == ERR_MSG_NOT_FOUND || err.Error() == ERR_RECIPIENT_NOT_FOUND {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		ctx.Logger.Error("ForwardMessage failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "database error"})
		return
	}

	// 6. OK
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(msg)
}
