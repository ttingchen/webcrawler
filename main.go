package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

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
	wg := sync.WaitGroup{}
	fmt.Println("Enter crawl")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.ParseForm()
	for k, v := range r.Form {
		ctx, cancel := context.WithCancel(r.Context())
		shutdown(ctx, func() {
			cancel()
			wg.Wait()
			log.Fatal("Graceful shutdown")
		})

		wg.Add(1)
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		prodname := strings.Join(v, "")
		if prodname == "" {
			w.WriteHeader(http.StatusBadRequest)
			wg.Done()
			return
		}
		w.WriteHeader(http.StatusOK)

		searchResult, err := crawl.SearchWeb(ctx, url.QueryEscape(prodname), w, r)
		if errors.Is(err, context.Canceled) {
			log.Println("User leave:", err)
		}
		if err != nil {
			log.Println("Unexpected errors: ", err)
			wg.Done()
			return
		}
		if err := crawl.LogResults(ctx, searchResult); err != nil {
			log.Println("Failed to log results:", err)
		}
		wg.Done()
	}
}

func shutdown(ctx context.Context, f func()) {
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)

		select {
		case <-c:
			f()
		}
	}()
}
