package api

import (
	"encoding/json"
	"net/http"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type updateNameRequest struct {
	NewName string `json:"newName"`
}

func (rt *_router) setMyUserName(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// Get userId from bearer token
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// Parse JSON
	var body updateNameRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid JSON",
		})
		return
	}

	// Validate name
	if len(body.NewName) < 3 || len(body.NewName) > 32 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid username length",
		})
		return
	}

	// Validate pattern ^[a-zA-Z0-9]+$
	// Optimization: Use package-level regex from forward-message.go
	if !fwdIDRegex.MatchString(body.NewName) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid username format",
		})
		return
	}

	// Check if username already exists
	taken, err := rt.db.IsUserNameTaken(body.NewName)
	if err != nil {
		ctx.Logger.Error("DB error in IsUserNameTaken: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	if taken {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "username already in use",
		})
		return
	}

	// Update name
	err = rt.db.UpdateUserName(userID, body.NewName)
	if err != nil {
		ctx.Logger.Error("DB error in UpdateUserName: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
