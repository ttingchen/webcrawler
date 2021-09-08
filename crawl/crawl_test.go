package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestSearchWeb(t *testing.T) {
	tests := []struct {
		name     string
		prodName string
	}{
		{
			name:     "search 100",
			prodName: "100",
		},
		{
			name:     "search apple",
			prodName: "apple",
		},
		{
			name:     "search cup",
			prodName: "cup",
		},
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/search-check", nil)
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	for _, tt := range tests {
		rr := httptest.NewRecorder()

		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				searchResult, err := SearchWeb(ctx, url.QueryEscape(tt.prodName), w, r)
				if err != nil || len(searchResult) == 0 {
					t.Fatalf("Search failed with err: %v and search length: %d", err, len(searchResult))
				}
				t.Log("Length of search results:", len(searchResult))
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
		})
	}

}

func TestLogResults(t *testing.T) {
	type args struct {
		ctx          context.Context
		searchResult []string
	}
	testctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	cancel()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "normal",
			args: args{
				ctx: context.Background(),
				searchResult: []string{
					`{"Name":"1997 Donruss Baseball Card Pick 1-250","Price":"NT$ 27","Image":"https://i.ebayimg.com/thumbs/images/g/urcAAOSwkkRfgQ2r/s-l225.jpg","URL":"https://www.ebay.com/itm/324327239886"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "context timeout",
			args: args{
				ctx: testctx,
				searchResult: []string{
					`{"Name":"1997 Donruss Baseball Card Pick 1-250","Price":"NT$ 27","Image":"https://i.ebayimg.com/thumbs/images/g/urcAAOSwkkRfgQ2r/s-l225.jpg","URL":"https://www.ebay.com/itm/324327239886"}`,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Log("test: ", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			if err := LogResults(tt.args.ctx, tt.args.searchResult); (err != nil) != tt.wantErr {
				t.Errorf("LogResults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
