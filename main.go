package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
func collectWatsons(prodname string) error {
	// number of the products
	count := 0

	// products per page
	prodPerPage := 32

	// the needed pages
	maxPageNum := maxProdNum / prodPerPage

	// check whether there is product in the page
	isElement := false

	// record the colly error
	Err := ""
	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 1 * time.Second,
		// Add an additional random delay
		RandomDelay: 5 * time.Second,

		Parallelism: 3,
	})

	c.OnHTML("e2-product-list", func(e *colly.HTMLElement) {
		e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
			isElement = true
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
		//fmt.Println("UserAgent", r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36")
	})

	finished := make(chan bool)
	flag := false
	for i := 0; i < maxPageNum; i++ {
		_ = withContextFunc(context.Background(), func() {
			log.Println("cancel from ctrl+c event")
			flag = true
		})

		isElement = false
		Url := fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodname, i)
		if err := c.Visit(Url); err != nil {
			log.Println("Url err:", err)
		}
		if !isElement {
			//log.Println("No more element on page", i+1)
			//break
		}
		if flag {
			close(finished)
			<-finished
			log.Println("Game over")
		}
	}
	c.Wait()
	if Err != "" {
		return errors.New(Err)
	}
	return nil
} //testing test

// scrape product info from Ebay website
func collectEbay(search_item string) error {

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
		RandomDelay: 5 * time.Second,

		DomainGlob:  "*.ebay.*",
		Parallelism: 4})

	c.OnHTML("div[class='s-item__wrapper clearfix']", func(e *colly.HTMLElement) {
		if prodNum <= maxProdNum {
			//avoid to get a null item
			if e.ChildText("h3[class='s-item__title']") != "" {
				fmt.Printf("Ebay #%v\n", prodNum)
				fmt.Println("Name: ", e.ChildText("h3[class='s-item__title']"))
				//use regex to remove the useless part of prodlink
				prodLink := e.ChildAttr("a[class='s-item__link']", "href")
				re := regexp.MustCompile(`\?(.*)`)
				fmt.Println("ProdLink: ", re.ReplaceAllString(prodLink, ""))
				fmt.Println("ImageLink: ", e.ChildAttr("img[class='s-item__image-img']", "src"))
				fmt.Println("Price: ", e.ChildText("span[class='s-item__price']"))
				fmt.Println("")

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

	finished := make(chan bool)
	flag := false
	//load 1 to pageNum pages
	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {
		_ = withContextFunc(context.Background(), func() {
			log.Println("cancel from ctrl+c event")
			flag = true
		})

		visitUrl := "https://www.ebay.com/sch/i.html?_nkw=" + search_item + "&_ipg=25&_pgn=" + strconv.Itoa(pageNum)
		if prodNum <= maxProdNum {
			if err := c.Visit(visitUrl); err != nil {
				log.Println("Url err:", err)
			}
		} else {
			//if we have enough product info, don't load next page
			break
		}
		if flag {
			close(finished)
			<-finished
			log.Println("Game over")
		}
	}
	c.Wait()
	if Err != "" {
		return errors.New(Err)
	}
	return nil

}

func main() {
	prodname := "100"
	//fmt.Scanln(&prodname)
	prodname = url.QueryEscape(prodname)

	//start := time.Now()
	if err := collectWatsons(prodname); err != nil {
		log.Fatal("collect Watsons fail:", err)
	}
	//fmt.Println(time.Since(start))

	//start := time.Now()
	if err := collectEbay(prodname); err != nil {
		log.Fatal("collect Ebay fail:", err)
	}
	//fmt.Println(time.Since(start))

}
