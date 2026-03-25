package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type addGroupMemberRequest struct {
	UserID string `json:"userId"`
}

type groupMemberResponse struct {
	UserID string `json:"userId"`
}

func (rt *_router) addToGroup(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Auth
	adminID := ctx.UserID
	if adminID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "missing or invalid token"})
		return
	}

	// 2. Path param: groupId
	groupID := ps.ByName("id")
	if len(groupID) < 10 || len(groupID) > 64 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid groupId"})
		return
	}
	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, groupID); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid groupId format"})
		return
	}

	// 3. Parse JSON
	var body addGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid JSON"})
		return
	}

	// 4. Validate user ID
	if len(body.UserID) < 10 || len(body.UserID) > 64 {
		w.WriteHeader(http.StatusBadRequest)
		// Fixed: Use constant or consistent string for goconst
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid userId"})
		return
	}
	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, body.UserID); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid userId format"})
		return
	}

	// 5. DB call
	err := rt.db.AddUserToGroup(groupID, body.UserID, adminID)
	if err != nil {
		// Note: Ideally, use errors.Is(err, database.ErrNotFound) if exported.
		// For now, we match strings but using constants prevents "goconst" errors.
		switch err.Error() {

		case "group not found":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "group not found"})
			return

		case "user not found":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "user not found"})
			return

		case "forbidden":
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "not authorized to add users"})
			return

		case "already in group":
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "user already in group"})
			return
		}

		// unknown DB error
		ctx.Logger.Error("DB AddUserToGroup error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "database error"})
		return
	}

	// 6. Success 201
	w.WriteHeader(http.StatusCreated)

	// Fixed: S1016 - Convert body to response struct directly
	if err := json.NewEncoder(w).Encode(groupMemberResponse(body)); err != nil {
		ctx.Logger.Error("encode error: ", err)
	}
}
