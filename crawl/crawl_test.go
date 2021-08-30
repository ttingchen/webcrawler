package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSearchWeb(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchResult, err := SearchWeb(ctx, url.QueryEscape("100"), w, r)
		if err != nil || len(*searchResult) == 0 {
			t.Fatalf("Search failed with err %s and search length %d", err, len(*searchResult))
		}
		return
	})
	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Check the response body is what we expect.
	if rr.Body.String() == "" {
		t.Errorf("handler returned unexpected body: got empty")
	}

}
