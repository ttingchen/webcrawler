package main

import (
	"context"
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

func collectWatsons(prodname string) {
	c := colly.NewCollector(
	//colly.Debugger(&debug.LogDebugger{}),
	)
	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 1 * time.Second,
		// Add an additional random delay
		RandomDelay: 5 * time.Second,
	})

	count := 0

	c.OnHTML("e2-product-list", func(e *colly.HTMLElement) {
		e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
			count++
			fmt.Println("Name: ", e.ChildText(".productName"))
			link := "https://www.watsons.com.tw" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href")
			fmt.Println("ProdLink: ", link)
			fmt.Println("ImgLink: ", e.ChildAttr("img", "src"))
			fmt.Println("Price: ", e.ChildText(".productPrice"))
			fmt.Printf("Watsons count:%v\n\n", count)
		})
		fmt.Println("Total page: ", e.ChildText("e2-total-number-product"))

		//cant find total page
	})

	c.OnResponse(func(r *colly.Response) {
		//fmt.Println(r)
	})

	c.OnRequest(func(r *colly.Request) {
		//fmt.Println("UserAgent", r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36")
	})

	for i := 0; i < 10; i++ {
		Url := fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodname, i)
		if err := c.Visit(Url); err != nil {
			fmt.Println(err)
		}
	}
}

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

func collectEbay(search_item string) {

	//get the max number of products to calculate the max number of pages
	max_page_num := 1
	c_page := colly.NewCollector()
	c_page.Limit(&colly.LimitRule{DomainGlob: "*.ebay.*", Parallelism: 5})
	c_page.OnHTML("h1[class='srp-controls__count-heading']", func(e *colly.HTMLElement) {
		re_num := regexp.MustCompile("[^0-9]")
		//atoi return string_to_int, error
		max_prod_num, _ := strconv.Atoi(re_num.ReplaceAllString(e.ChildText("span[class='BOLD']"), ""))
		max_page_num = max_prod_num/25 + 1
	})
	c_page.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36")
	})
	visit_url := "https://www.ebay.com/sch/i.html?_nkw=" + search_item + "&_ipg=25"
	c_page.Visit(visit_url)

	prod_num := 1
	//we wanna see the number of prodect
	prod_num_set := 200
	//load 1 to page_num pages
	finished := make(chan bool)
	flag := false
	for page_num := 1; page_num <= max_page_num; page_num++ {
		_ = withContextFunc(context.Background(), func() {
			log.Println("cancel from ctrl+c event")
			flag = true

		})
		c := colly.NewCollector()
		c.Limit(&colly.LimitRule{DomainGlob: "*.ebay.*", Parallelism: 5})

		c.OnHTML("div[class='s-item__wrapper clearfix']", func(e *colly.HTMLElement) {
			if prod_num <= prod_num_set {
				//avoid to get a null item
				if e.ChildText("h3[class='s-item__title']") != "" {
					fmt.Println(prod_num, ".Name: ", e.ChildText("h3[class='s-item__title']"))
					//use regex to remove the useless part of prodlink
					prod_link := e.ChildAttr("a[class='s-item__link']", "href")
					re := regexp.MustCompile(`\?(.*)`)
					fmt.Println("ProdLink: ", re.ReplaceAllString(prod_link, ""))
					fmt.Println("ImageLink: ", e.ChildAttr("img[class='s-item__image-img']", "src"))
					fmt.Println("Price: ", e.ChildText("span[class='s-item__price']"))
					fmt.Println("")

					prod_num += 1
				}

			}
		})

		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36")
		})

		visit_url := "https://www.ebay.com/sch/i.html?_nkw=" + search_item + "&_ipg=25&_pgn=" + strconv.Itoa(page_num)
		if prod_num <= prod_num_set {
			c.Visit(visit_url)
		} else {
			break
		}
		if flag {
			close(finished)
			<-finished
			log.Println("Game over")
		}
	}

}

func main() {
	prodname := "monitor"
	//fmt.Scanln(&prodname)
	prodname = url.QueryEscape(prodname)

	//collectWatsons(prodname)
	collectEbay(prodname)

}
