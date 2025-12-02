package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

var fwdIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

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
	if len(body.MessageID) < 10 || len(body.MessageID) > 30 || !fwdIDRegex.MatchString(body.MessageID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid messageId"})
		return
	}

	// 4. Validate recipientId
	if len(body.RecipientID) < 10 || len(body.RecipientID) > 30 || !fwdIDRegex.MatchString(body.RecipientID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid recipientId"})
		return
	}

	// 5. Forward
	msg, err := rt.db.ForwardMessage(userID, body.MessageID, body.RecipientID)
	if err != nil {
		if err.Error() == "message not found" || err.Error() == "recipient not found" {
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
