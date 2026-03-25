package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type addReactionRequest struct {
	Emoticon string `json:"emoticon"`
}

func (rt *_router) commentMessage(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Extract messageId
	messageID := ps.ByName("id")
	if len(messageID) < 10 || len(messageID) > 64 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid message id"})
		return
	}

	// 2. Validate bearer auth
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 3. Parse JSON
	var body addReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid JSON"})
		return
	}

	// 4. Validate emoticon
	if len(body.Emoticon) < 1 || len(body.Emoticon) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid emoticon length"})
		return
	}

	// pattern ".+" from API
	if ok, _ := regexp.MatchString(`.+`, body.Emoticon); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid emoticon format"})
		return
	}

	// 5. DB call
	err := rt.db.AddReaction(messageID, userID, body.Emoticon)
	if err != nil {
		if err.Error() == "message not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "message not found"})
			return
		}

		ctx.Logger.Error("DB AddReaction error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "database error"})
		return
	}

	// 6. Success
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
