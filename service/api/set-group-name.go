package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type updateGroupNameRequest struct {
	NewName string `json:"newName"`
}

func (rt *_router) setGroupName(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Extract groupId from path
	groupID := ps.ByName("id")

	if len(groupID) < 10 || len(groupID) > 30 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid group id"})
		return
	}

	// 2. Auth: Only authenticated user allowed
	userID := ctx.UserID
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "missing or invalid token",
		})
		return
	}

	// 3. Parse body
	var body updateGroupNameRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid JSON"})
		return
	}

	// 4. Validate newName
	if len(body.NewName) < 3 || len(body.NewName) > 32 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid group name length"})
		return
	}

	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9 ]+$`, body.NewName); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid group name format"})
		return
	}

	// 5. Call DB
	err := rt.db.SetGroupName(groupID, userID, body.NewName)

	if err != nil {
		// FIXED: Use package-level constants to satisfy goconst
		switch err.Error() {

		case ERR_FORBIDDEN:
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "not allowed to rename group",
			})
			return

		case ERR_GROUP_NOT_FOUND:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "group not found",
			})
			return

		case "name exists": // API -> 409
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "name already in use",
			})
			return
		}

		// Unexpected DB error
		ctx.Logger.Error("DB error in SetGroupName: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// 6. Success
	w.WriteHeader(http.StatusNoContent)
}
