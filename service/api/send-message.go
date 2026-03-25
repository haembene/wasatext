package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

var idRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

type sendMessageRequest struct {
	RecipientID string  `json:"recipientId"`
	Content     string  `json:"content"`
	ReplyTo     *string `json:"replyTo"`
}

func (rt *_router) sendMessage(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Auth
	senderID := ctx.UserID
	if senderID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 2. Parse body
	var body sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid JSON"})
		return
	}

	// 3. Validate recipientId
	if len(body.RecipientID) < 3 || len(body.RecipientID) > 64 || !idRegex.MatchString(body.RecipientID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid recipientId"})
		return
	}

	// 4. Validate content (Boosted to 10MB to support Base64 Inline Images)
	if len(body.Content) < 1 || len(body.Content) > 10000000 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid content"})
		return
	}

	// 5. Validate replyTo
	if body.ReplyTo != nil {
		if len(*body.ReplyTo) < 3 || len(*body.ReplyTo) > 64 || !idRegex.MatchString(*body.ReplyTo) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid replyTo"})
			return
		}
	}

	// 6. DB call
	msg, err := rt.db.SendMessage(senderID, body.RecipientID, body.Content, body.ReplyTo)
	if err != nil {
		// Use constant defined in forward-message.go to satisfy goconst
		if err.Error() == ERR_RECIPIENT_NOT_FOUND {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "recipient not found"})
			return
		}

		ctx.Logger.Error("SendMessage failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "database error"})
		return
	}

	// 7. Success
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		ctx.Logger.Error("encode error: ", err)
	}
}
