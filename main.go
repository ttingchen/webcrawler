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
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)

		select {
		case <-c:
			f()
		}
	}()

	return ctx
}

// scrape product info from Watsons website
func collectWatsons(ctx context.Context, prodname string) error {
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
		select {
		case <-ctx.Done():
			log.Println("Quit from watsons collector")
			c.Wait()
			return nil
		default:
			time.Sleep(2 * time.Second)
			Url := fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodname, i)
			if err := c.Visit(Url); err != nil {
				log.Println("Url err:", err)
			}
		}
	}
	c.Wait()

	if Err != "" {
		return errors.New(Err)
	}
	return nil
}

// scrape product info from Ebay website
func collectEbay(ctx context.Context, searchItem string) error {

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

	//load 1 to pageNum pages
	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {
		select {
		case <-ctx.Done():
			log.Println("Quit from Ebay collector")
			c.Wait()
			return nil
		default:
			time.Sleep(1 * time.Second)
			visitUrl := "https://www.ebay.com/sch/i.html?_nkw=" + searchItem + "&_ipg=25&_pgn=" + strconv.Itoa(pageNum)
			if prodNum <= maxProdNum {
				if err := c.Visit(visitUrl); err != nil {
					log.Println("Url err:", err)
				}
			} else {
				// If we have enough product info, don't load next page.
				break
			}
		}
	}
	c.Wait()
	if Err != "" {
		return errors.New(Err)
	}
	return nil

}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	// for graceful shut down
	_ = withContextFunc(ctx, func() {
		log.Println("cancel from ctrl+c event")
		cancel()
	})

	prodname := "100"
	//fmt.Scanln(&prodname)
	prodname = url.QueryEscape(prodname)

	//start := time.Now()
	go collectWatsons(ctx, prodname)

	if err := collectEbay(ctx, prodname); err != nil {
		log.Fatal("collect Ebay fail:", err)
	}

	// fmt.Println(time.Since(start))

}
