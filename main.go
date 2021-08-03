package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/gocolly/colly"
)

func collectWatsons(prodname string) {
	c := colly.NewCollector(
		//colly.Debugger(&debug.LogDebugger{}),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		// Delay: 1 * time.Second,
		// // Add an additional random delay
		// RandomDelay: 5 * time.Second,
		Parallelism: 2,
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

		//cant find total page
	})

	c.OnResponse(func(r *colly.Response) {
		//fmt.Println(r)
	})

	c.OnRequest(func(r *colly.Request) {
		//fmt.Println("UserAgent", r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36")
	})

	for i := 0; i < 2; i++ {
		Url := fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodname, i)
		//fmt.Println(isValidUrl("Url"))
		if err := c.Visit(Url); err != nil {
			fmt.Println("err:", err)
		}
	}
	c.Wait()
}

func collectEbay(search_item string) {

	prod_num := 1
	//load 1 to page_num pages
	for page_num := 1; page_num <= 10; page_num++ {
		c := colly.NewCollector()
		c.Limit(&colly.LimitRule{DomainGlob: "*.ebay.*", Parallelism: 5})

		c.OnHTML("div[class='s-item__wrapper clearfix']", func(e *colly.HTMLElement) {
			if prod_num <= 30 {
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

		visit_url := "https://www.ebay.com/sch/i.html?_nkw=" + search_item + "&_pgn=" + strconv.Itoa(page_num)

		c.Visit(visit_url)
	}
}

func main() {
	prodname := "指甲貼"
	//fmt.Scanln(&prodname)
	prodname = url.QueryEscape(prodname)

	collectWatsons(prodname)
	//collectEbay(prodname)

}
