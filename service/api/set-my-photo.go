package api

import (
	"encoding/json"
	"net/http"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type updatePhotoRequest struct {
	PhotoUrl string `json:"photoUrl"`
}

func (rt *_router) setMyPhoto(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Get userId from bearer token
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 2. Parse JSON body
	var body updatePhotoRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid JSON",
		})
		return
	}

	// 3. Validate photoUrl
	if len(body.PhotoUrl) == 0 || len(body.PhotoUrl) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid photo url",
		})
		return
	}

	// Validate format (using package-level variable from set-group-photo.go)
	if !urlRegex.MatchString(body.PhotoUrl) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid url format",
		})
		return
	}

	// 4. Update DB
	err := rt.db.UpdateUserPhoto(userID, body.PhotoUrl)
	if err != nil {
		ctx.Logger.Error("DB error in UpdateUserPhoto: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 5. Return 204
	w.WriteHeader(http.StatusNoContent)
}
