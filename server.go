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
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gocolly/colly"
)

// max product amount for each online store
const maxProdNum = 500

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

// scrape product info from Watsons website
func collectWatsons(w http.ResponseWriter, prodname string, flag *bool) error {
	// number of the products
	count := 0

	// products per page
	prodPerPage := 32

	// the needed pages
	maxPageNum := maxProdNum / prodPerPage

	// record the colly error
	Err := ""
	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 1 * time.Second,
		// Add an additional random delay
		RandomDelay: 3 * time.Second,

		Parallelism: 3,
	})

	c.OnHTML("e2-product-list", func(e *colly.HTMLElement) {
		e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
			count++
			fmt.Printf("Watsons #%v\n", count)
			fmt.Println("Name: ", e.ChildText(".productName"))
			link := "https://www.watsons.com.tw" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href")
			fmt.Println("ProdLink: ", link)
			fmt.Println("ImgLink: ", e.ChildAttr("img", "src"))
			fmt.Println("Price: ", e.ChildText(".productPrice"))
			fmt.Println("")
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		Err = fmt.Sprintln("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36")
	})

	for i := 0; i < maxPageNum; i++ {
		if *flag {
			log.Println("Quit from watsons collector")
			return nil
		}

		Url := fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodname, i)
		if err := c.Visit(Url); err != nil {
			log.Println("Url err:", err)
		}
	}
	c.Wait()

	if Err != "" {
		return errors.New(Err)
	}
	return nil
}

// scrape product info from Ebay website
func collectEbay(w http.ResponseWriter, search_item string, flag *bool) error {

	Err := ""
	prodNum := 1
	maxPageNum := maxProdNum / 25

	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 1 * time.Second,
		// Add an additional random delay
		RandomDelay: 3 * time.Second,

		DomainGlob:  "*.ebay.*",
		Parallelism: 10,
	})

	c.OnHTML("div[class='s-item__wrapper clearfix']", func(e *colly.HTMLElement) {
		if prodNum <= maxProdNum {
			//avoid to get a null item
			if e.ChildText("h3[class='s-item__title']") != "" {
				fmt.Fprintf(w, "Ebay #%v\n", prodNum)
				fmt.Fprintf(w, "Name: %v\n", e.ChildText("h3[class='s-item__title']"))
				//use regex to remove the useless part of prodlink
				prodLink := e.ChildAttr("a[class='s-item__link']", "href")
				re := regexp.MustCompile(`\?(.*)`)
				fmt.Fprintf(w, "ProdLink: %v\n", re.ReplaceAllString(prodLink, ""))
				fmt.Fprintf(w, "ImageLink: %v\n", e.ChildAttr("img[class='s-item__image-img']", "src"))
				fmt.Fprintf(w, "Price: %v\n\n", e.ChildText("span[class='s-item__price']"))

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

func collyCrawler(w http.ResponseWriter, r *http.Request) {
	// for graceful shut down
	flag := false
	_ = withContextFunc(context.Background(), func() {
		log.Println("cancel from ctrl+c event")
		flag = true
	})

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		prodname := strings.Join(v, "")
		if err := collectEbay(w, url.QueryEscape(prodname), &flag); err != nil {
			log.Fatal("collect Ebay fail:", err)
		}
	}
}

func main() {

	//usage: http://localhost:9090/?url_long=keyword
	http.HandleFunc("/", collyCrawler)
	//set port number
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
