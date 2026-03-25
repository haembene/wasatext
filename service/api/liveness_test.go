package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestLiveness(t *testing.T) {
	// Create a minimal router struct testing strictly the liveness logic
	rt := &_router{}

	// Construct exactly the HTTP request the handler expects
	req, err := http.NewRequest("GET", "/liveness", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Direct call to the actual handler method
	rt.liveness(rr, req, httprouter.Params{})

	// Validate results against expectations: HTTP 204 No Content
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}
