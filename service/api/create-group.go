package api

import (
	"encoding/json"
	"net/http"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// createGroup handles POST /groups
func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. Parse the JSON body
	var reqBody struct {
		Name string `json:"name"`
	}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// 2. Validate input
	if len(reqBody.Name) < 3 || len(reqBody.Name) > 30 {
		http.Error(w, "Group name must be between 3 and 30 chars", http.StatusBadRequest)
		return
	}

	// 3. Call the database function
	group, err := rt.db.CreateGroup(reqBody.Name, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error creating group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. Return the new group as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(group)
}
