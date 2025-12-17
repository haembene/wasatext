package api

import (
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"github.com/haembene/wasatext/service/api/reqcontext"
)

func (rt *_router) setConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	
	targetUserID := ps.ByName("id")

	conversation, err := rt.db.CreateConversation(ctx.UserID, targetUserID)
	
	if err != nil {
		ctx.Logger.WithError(err).Error("Can't create conversation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	_ = json.NewEncoder(w).Encode(conversation)
}