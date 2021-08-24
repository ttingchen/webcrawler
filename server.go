package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/gocolly/colly"
)

// max product amount for each online store
const (
	maxProdNum = 500
)

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
	ctx := r.Context()
	prodname := r.URL.Query().Get("search")
	if prodname == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	searchResult, err := collectEbay(ctx, w, r, url.QueryEscape(prodname))
	if err != nil {
		log.Fatal("collect Ebay fail:", err)
	}
	for i := 1; i <= maxProdNum; i++ {
		json.NewDecoder(r.Body).Decode(&searchResult[i])
		fmt.Println("Ebay #", i, ": ")
		fmt.Println(searchResult[i].Name)
		fmt.Println(searchResult[i].URL)
		fmt.Println(searchResult[i].Image)
		fmt.Println(searchResult[i].Price)
		fmt.Println()
	}

}

// scrape product info from Ebay website
func collectEbay(ctx context.Context, w http.ResponseWriter, r *http.Request, search_item string) (*[maxProdNum + 100]Product, error) {

	prodNum := 1
	prodPerPage := 25
	maxPageNum := maxProdNum / prodPerPage
	var result [maxProdNum + 100]Product

	Err := ""
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36"),
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

				fmt.Fprintf(w, "#%v: json.NewEncode:\n", prodNum)

				result[prodNum] = Product{prodName, prodPrice, prodImgLink, prodLinkR}
				if err := json.NewEncoder(w).Encode(&result[prodNum]); err == nil {
					fmt.Fprintf(w, "")
				}
				fmt.Fprintf(w, "\n")

				prodNum++
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		Err = fmt.Sprintln("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnRequest(func(r *colly.Request) {
		select {
		case <-ctx.Done(): // 如果 canceled
			r.Abort() // 結束 request
		default: // 要有 default，不然 select {} 會卡住
		}
	})

	//load 1 to pageNum pages
	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {

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
		return nil, errors.New(Err)
	}
	return &result, nil
}

type Product struct {
	Name  string `json:"name"`
	Price string `json:"price"`
	Image string `json:"image_link"`
	URL   string `json:"url"`
}

type webUtil interface {
	onHTMLFunc(e *colly.HTMLElement, prodNum, maxProdNum int, w http.ResponseWriter)
	getURL(prodName string, pageNum int) string
	getInfo() webInfo
}

type webInfo struct {
	Name       string
	NumPerPage int
	OnHTML     string
}

type watsonsUtil webInfo
type ebayUtil webInfo

func (u *ebayUtil) onHTMLFunc(e *colly.HTMLElement, prodNum, maxProdNum int, w http.ResponseWriter) {
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
}

func (u *ebayUtil) getURL(prodName string, pageNum int) string {
	return "https://www.ebay.com/sch/i.html?_nkw=" + prodName + "&_ipg=25&_pgn=" + strconv.Itoa(pageNum)
}

func (u *ebayUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
	}
}

func (u *watsonsUtil) onHTMLFunc(e *colly.HTMLElement, prodNum, maxProdNum int, w http.ResponseWriter) {
	e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
		prodNum++
		fmt.Printf("Watsons #%v\n", prodNum)
		fmt.Println("Name: ", e.ChildText(".productName"))
		link := "https://www.watsons.com.tw" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href")
		fmt.Println("ProdLink: ", link)
		fmt.Println("ImgLink: ", e.ChildAttr("img", "src"))
		fmt.Println("Price: ", e.ChildText(".productPrice"))
		fmt.Println("")
	})
}

func (u *watsonsUtil) getURL(prodName string, pageNum int) string {
	return fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&currentPage=%d", prodName, pageNum)
}

func (u *watsonsUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
	}
}

func searchWeb(ctx context.Context, prodName string, w http.ResponseWriter) error {

	var ebayInfo webUtil = &ebayUtil{
		Name:       "Ebay",
		NumPerPage: 25,
		OnHTML:     "div[class='s-item__wrapper clearfix']",
	}
	var watsonInfo webUtil = &watsonsUtil{
		Name:       "Watsons",
		NumPerPage: 32,
		OnHTML:     "e2-product-list",
	}

	websites := []webUtil{
		ebayInfo,
		watsonInfo,
	}

	for _, website := range websites {
		crawlWebsite(ctx, website, prodName, w)
	}

	return nil
}

// scrape product info from website
func crawlWebsite(ctx context.Context, webutil webUtil, prodName string, w http.ResponseWriter) error {
	Err := ""
	prodNum := 1
	webinfo := webutil.getInfo()

	maxPageNum := maxProdNum / webinfo.NumPerPage

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

	c.OnHTML(webinfo.OnHTML, func(e *colly.HTMLElement) {
		// for each website
		webutil.onHTMLFunc(e, prodNum, maxProdNum, w)
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
			log.Println("Quit from collector: ")
			c.Wait()
			return nil
		default:
			time.Sleep(1 * time.Second)
			visitURL := webutil.getURL(prodName, pageNum)
			if prodNum <= maxProdNum {
				if err := c.Visit(visitURL); err != nil {
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