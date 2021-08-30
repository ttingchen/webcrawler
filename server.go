package main

import (
	"Go_WebService/crawl"
	"context"
	"encoding/json"
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
	ctx := r.Context()

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		prodname := strings.Join(v, "")
		if prodname == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

		searchResult, err := crawl.SearchWeb(ctx, url.QueryEscape(prodname), w, r)
		if err != nil {
			log.Fatal("Failed to search: ", err)
		}

		if err := logResults(ctx, searchResult); err != nil {
			log.Println("Failed to log results:", err)
		}
	}
}

func logResults(ctx context.Context, searchResult *[]string) error {
	fmt.Println("Start to log results")
	for i, result := range *searchResult {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		var product crawl.Product
		if err := json.NewDecoder(strings.NewReader(result)).Decode(&product); err == nil {
			fmt.Printf("Total #%d : \n%v\n%v\n%v\n%v\n\n", i+1, product.Name, product.URL, product.Image, product.Price)
		} else {
			return err
		}
	}
	return nil
}
