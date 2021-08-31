package main

import (
	"Go_WebService/crawl"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
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
		if errors.Is(err, context.Canceled) { // 若用戶中離，結束
			log.Println("User leave:", err)
			log.Println("Searched len:", len(*searchResult))
			return
		}
		if err != nil {
			log.Fatal("Unexpected errors: ", err)
		}

		if err := crawl.LogResults(ctx, searchResult); err != nil {
			log.Println("Failed to log results:", err)
		}
	}
}
