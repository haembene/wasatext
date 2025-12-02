package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) findUsers(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Authorization: must have authenticated user
	if ctx.UserID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 2. Extract query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "username parameter required",
		})
		return
	}

	// 3. Length validation
	if len(username) < 3 || len(username) > 32 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid username length",
		})
		return
	}

	// 4. Pattern validation
	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(username) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "username contains invalid characters",
		})
		return
	}

	// 5. DB lookup
	users, err := rt.db.FindUsers(username)
	if err != nil {
		ctx.Logger.Error("FindUsers failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 6. Success
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(users)
}
