package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/gocolly/colly"
)

// max product amount for each online store
const maxProdNum = 500

func main() {
	//usage: http://localhost:9090/?search=keyword
	http.HandleFunc("/", collyCrawler)
	//set port number
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func collyCrawler(w http.ResponseWriter, r *http.Request) {
	// for graceful shut down
	flag := false
	_ = withContextFunc(context.Background(), func() {
		log.Println("cancel from ctrl+c event")
		flag = true
	})

	prodname := r.URL.Query().Get("search")
	err := collectEbay(w, r, url.QueryEscape(prodname), &flag)
	if err != nil {
		log.Fatal("collect Ebay fail:", err)
	}

}

//check if there is ctrl+c
func withContextFunc(ctx context.Context, f func()) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
			f()
		}
	}()

	return ctx
}

// scrape product info from Ebay website
func collectEbay(w http.ResponseWriter, r *http.Request, search_item string, flag *bool) error {

	prodNum := 1

	prodPerPage := 25
	maxPageNum := maxProdNum / prodPerPage

	Err := ""
	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 1 * time.Second,
		// Add an additional random delay
		RandomDelay: 3 * time.Second,

		// DomainGlob:  "*.ebay.*",
		Parallelism: 3,
	})

	c.OnHTML("div[class='s-item__wrapper clearfix']", func(e *colly.HTMLElement) {
		if prodNum <= maxProdNum {
			//avoid to get a null item
			if e.ChildText("h3[class='s-item__title']") != "" {
				//use regex to remove the useless part of prodlink
				re := regexp.MustCompile(`\?(.*)`)
				prodName := e.ChildText("h3[class='s-item__title']")
				prodLink := e.ChildAttr("a[class='s-item__link']", "href")
				prodLinkR := re.ReplaceAllString(prodLink, "")
				prodImgLink := e.ChildAttr("img[class='s-item__image-img']", "src")
				prodPrice := e.ChildText("span[class='s-item__price']")

				// fmt.Fprintf(w, "Ebay #%v\n", prodNum)
				// fmt.Fprintf(w, "Name: %v\n", prodName)
				// fmt.Fprintf(w, "ProdLink: %v\n", prodLinkR)
				// fmt.Fprintf(w, "ImageLink: %v\n", prodImgLink)
				// fmt.Fprintf(w, "Price: %v\n", prodPrice)
				fmt.Fprintf(w, "#%v: json.NewEncode:\n", prodNum)

				result := Product{prodName, prodPrice, prodImgLink, prodLinkR}
				if err := json.NewEncoder(w).Encode(&result); err == nil {
					fmt.Fprintf(w, "")
				}
				fmt.Fprintf(w, "\n")

				json.NewDecoder(r.Body).Decode(&result)
				fmt.Println("Ebay #", prodNum, ": ")
				fmt.Println(result.Name)
				fmt.Println(result.URL)
				fmt.Println(result.Image)
				fmt.Println(result.Price)
				fmt.Println()

				prodNum++
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		Err = fmt.Sprintln("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36")
	})

	//load 1 to pageNum pages
	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {
		if *flag {
			log.Println("Quit from ebay collector")
			return nil
		}

		visitUrl := "https://www.ebay.com/sch/i.html?_nkw=" + search_item + "&_ipg=25&_pgn=" + strconv.Itoa(pageNum)
		if prodNum <= maxProdNum {
			if err := c.Visit(visitUrl); err != nil {
				log.Println("Url err:", err)
			}
		} else {
			//if we have enough product info, don't load next page
			break
		}
	}
	c.Wait()
	if Err != "" {
		return errors.New(Err)
	}
	return nil

}

type Product struct {
	Name  string `json:"name"`
	Price string `json:"price"`
	Image string `json:"image_link"`
	URL   string `json:"url"`
}
