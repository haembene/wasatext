package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) uncommentMessage(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Extract messageId
	messageID := ps.ByName("id")
	if len(messageID) < 10 || len(messageID) > 30 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid message id",
		})
		return
	}

	// 2. Validate token
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 3. Get emoticon from query
	emoticon := r.URL.Query().Get("emoticon")
	if emoticon == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "emoticon is required",
		})
		return
	}

	// 4. Validate emoticon length and pattern
	if len(emoticon) < 1 || len(emoticon) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid emoticon length",
		})
		return
	}

	if ok, _ := regexp.MatchString(`.+`, emoticon); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid emoticon format",
		})
		return
	}

	// 5. DB call
	err := rt.db.RemoveReaction(messageID, userID, emoticon)
	if err != nil {
		if err.Error() == "message not found" || err.Error() == "reaction not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": err.Error(),
			})
			return
		}

		ctx.Logger.Error("DB RemoveReaction error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 6. Success
	w.WriteHeader(http.StatusNoContent)
}
