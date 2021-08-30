package crawl

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSearchWeb(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/search-check", nil)
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		searchResult, err := SearchWeb(ctx, url.QueryEscape(""), w, r)
		if err != nil || len(*searchResult) == 0 {
			t.Fatalf("Search failed with err: %s and search length: %d", err, len(*searchResult))
		}
		t.Log("Length of search results:", len(*searchResult))
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
