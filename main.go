package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/ttingchen/webcrawler/crawl"
)

func main() {
	//usage: http://localhost:9090/search?keyword=100
	http.HandleFunc("/search", collyCrawler)
	//set port number
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func collyCrawler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Enter crawl")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.ParseForm()
	for k, v := range r.Form {
		ctx := r.Context()
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		prodname := strings.Join(v, "")
		if prodname == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

		searchResult, err := crawl.SearchWeb(ctx, url.QueryEscape(prodname), w, r)
		if errors.Is(err, context.Canceled) {
			log.Println("User leave:", err)
		}
		if err != nil {
			log.Println("Unexpected errors: ", err)
			return
		}
		if err := crawl.LogResults(ctx, searchResult); err != nil {
			log.Println("Failed to log results:", err)
		}
	}
}
