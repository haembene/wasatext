package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) leaveGroup(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Authentication
	authUser := ctx.UserID
	if authUser == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 2. Extract path params
	groupID := ps.ByName("groupId")
	userID := ps.ByName("userId")

	// 3. userId must be the authenticated user
	if userID != authUser {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "cannot remove another user",
		})
		return
	}

	// 4. Validate groupId and userId format
	if !isValidID(groupID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid groupId"})
		return
	}

	if !isValidID(userID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid userId"})
		return
	}

	// 5. Call DB
	err := rt.db.LeaveGroup(groupID, userID, authUser)
	if err != nil {

		switch err.Error() {
		case "group not found":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "group not found"})
			return

		case "user not found":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "user not found"})
			return

		case "not in group":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "user is not a member of this group",
			})
			return

		case "forbidden":
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "cannot remove another user",
			})
			return

		default:
			ctx.Logger.Error("DB LeaveGroup error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "database error"})
			return
		}
	}

	// 6. Success
	w.WriteHeader(http.StatusNoContent)
}

func isValidID(id string) bool {
	if len(id) < 10 || len(id) > 30 {
		return false
	}
	ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, id)
	return ok
}
