package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/haembene/wasatext/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

type loginRequest struct {
	Name string `json:"name"`
}

type authResponse struct {
	Identifier string `json:"identifier"`
}

func (rt *_router) doLogin(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	ctx reqcontext.RequestContext,
) {
	w.Header().Set("Content-Type", "application/json")

	// Parse JSON
	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid JSON",
		})
		return
	}

	// Validate username exists
	if body.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "name is required",
		})
		return
	}

	// Validate length
	if len(body.Name) < 3 || len(body.Name) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid username length",
		})
		return
	}

	// Validate pattern ^[a-zA-Z0-9]+$
	if ok, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, body.Name); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid username format",
		})
		return
	}

	// Login (or create)
	userID, err := rt.db.LoginOrCreateUser(body.Name)
	if err != nil {
		ctx.Logger.Error("LoginOrCreateUser failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "database error",
		})
		return
	}

	// Response
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(authResponse{
		Identifier: userID,
	}); err != nil {
		ctx.Logger.Error("encode error: ", err)
	}
}
