package api

import (
	"encoding/json"
	"net/http"


	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)



type updateGroupPhotoRequest struct {
	PhotoUrl string `json:"photoUrl"`
}

func (rt *_router) setGroupPhoto(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Parse group ID
	groupID := ps.ByName("id")
	if len(groupID) < 10 || len(groupID) > 64 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid group id"})
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

	// 3. Parse JSON body
	var body updateGroupPhotoRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid JSON",
		})
		return
	}

	// 4. Validate photo URL bounds implicitly allocating indefinitely large File payload data resolving inline base64 formats.
	if len(body.PhotoUrl) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid photo url length",
		})
		return
	}

	// 5. Database call
	err := rt.db.SetGroupPhoto(groupID, userID, body.PhotoUrl)

	if err != nil {
		switch err.Error() {

		case ERR_FORBIDDEN:
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "not allowed to change group photo",
			})
			return

		case ERR_GROUP_NOT_FOUND:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "group not found",
			})
			return
		}

		ctx.Logger.Error("DB SetGroupPhoto error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 6. Success
	w.WriteHeader(http.StatusNoContent)
}
