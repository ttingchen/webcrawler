package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

	requestBody, _ := ioutil.ReadAll(r.Body)
	r.ContentLength = int64(len(string(requestBody)))
	r.TransferEncoding = []string{"identity"}

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		prodname := strings.Join(v, "")
		if prodname == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		searchResult, err := searchWeb(ctx, url.QueryEscape(prodname), w, r)
		if err != nil {
			log.Fatal("collect Ebay fail:", err)
		}
		for i, result := range *searchResult {
			json.NewDecoder(r.Body).Decode(&result)
			fmt.Println("Total #", i, ": ")
			fmt.Println(result.Name)
			fmt.Println(result.URL)
			fmt.Println(result.Image)
			fmt.Println(result.Price)
			fmt.Println()
		}
	}
}

// Product is product
type Product struct {
	Name  string `json:"name"`
	Price string `json:"price"`
	Image string `json:"image_link"`
	URL   string `json:"url"`
}

type webUtil interface {
	onHTMLFunc(e *colly.HTMLElement, prodNum *int, w http.ResponseWriter, result *[]Product) error
	getURL(prodName string, pageNum int) string
	getInfo() webInfo
}

type webInfo struct {
	Name       string
	NumPerPage int
	OnHTML     string
	UserAgent  string
}

type watsonsUtil webInfo
type ebayUtil webInfo

func (u *ebayUtil) onHTMLFunc(e *colly.HTMLElement, prodNum *int, w http.ResponseWriter, result *[]Product) (err error) {
	num := *prodNum
	buf := new(bytes.Buffer)
	if num <= maxProdNum {
		//avoid to get a null item
		if e.ChildText("h3[class='s-item__title']") != "" {
			//use regex to remove the useless part of prodlink
			re := regexp.MustCompile(`\?(.*)`)
			prodName := e.ChildText("h3[class='s-item__title']")
			prodLink := e.ChildAttr("a[class='s-item__link']", "href")
			prodLinkR := re.ReplaceAllString(prodLink, "")
			prodImgLink := e.ChildAttr("img[class='s-item__image-img']", "src")
			prodPrice := e.ChildText("span[class='s-item__price']")

			fmt.Fprintf(w, "Ebay #%v: json.NewEncode:\n", num)

			*result = append(*result, Product{prodName, prodPrice, prodImgLink, prodLinkR})
			n := len(*result)
			if err = json.NewEncoder(buf).Encode(&(*result)[n-1]); err != nil {
				fmt.Println(err)
			} else {
				io.Copy(w, buf)
				fmt.Fprintf(w, "\n")
			}

			*prodNum = num + 1
		}
	}
	return err
}

func (u *ebayUtil) getURL(prodName string, pageNum int) string {
	return "https://www.ebay.com/sch/i.html?_nkw=" + prodName + "&_ipg=25&_pgn=" + strconv.Itoa(pageNum)
}

func (u *ebayUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
		UserAgent:  u.UserAgent,
	}
}

func (u *watsonsUtil) onHTMLFunc(e *colly.HTMLElement, prodNum *int, w http.ResponseWriter, result *[]Product) (err error) {
	num := *prodNum
	buf := new(bytes.Buffer)
	e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
		prodName := e.ChildText(".productName")
		prodLink := "https://www.watsons.com.tw" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href")
		prodImgLink := e.ChildAttr("img", "src")
		prodPrice := e.ChildText(".productPrice")
		fmt.Fprintf(w, "Watsons #%v: json.NewEncode:\n", num)

		*result = append(*result, Product{prodName, prodPrice, prodImgLink, prodLink})
		n := len(*result)
		if err = json.NewEncoder(buf).Encode(&(*result)[n-1]); err != nil {
			fmt.Println(err)
			return
		} else {
			io.Copy(w, buf)
			fmt.Fprintf(w, "\n")
		}
		num++
	})
	*prodNum = num

	return err
}

func (u *watsonsUtil) getURL(prodName string, pageNum int) string {
	return fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&pageSize=64&currentPage=%d", prodName, pageNum)
}

func (u *watsonsUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
		UserAgent:  u.UserAgent,
	}
}

func searchWeb(ctx context.Context, prodName string, w http.ResponseWriter, r *http.Request) (*[]Product, error) {
	var ebayInfo webUtil = &ebayUtil{
		Name:       "Ebay",
		NumPerPage: 25,
		OnHTML:     "div[class='s-item__wrapper clearfix']",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36",
	}
	var watsonInfo webUtil = &watsonsUtil{
		Name:       "Watsons",
		NumPerPage: 64,
		OnHTML:     "e2-product-list",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36",
	}

	websites := []webUtil{
		ebayInfo,
		watsonInfo,
	}

	var result []Product

	for _, website := range websites {

		err := crawlWebsite(ctx, website, prodName, &result, w, r)
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// scrape product info from website
func crawlWebsite(ctx context.Context, webutil webUtil, prodName string, result *[]Product, w http.ResponseWriter, r *http.Request) error {
	Err := ""
	prodNum := 1
	webinfo := webutil.getInfo()

	maxPageNum := maxProdNum / webinfo.NumPerPage

	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent(webinfo.UserAgent),
	)
	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 3 * time.Second,
		// Add an additional random delay
		RandomDelay: 5 * time.Second,

		Parallelism: 3,
	})

	c.OnHTML(webinfo.OnHTML, func(e *colly.HTMLElement) {
		// for each website
		webutil.onHTMLFunc(e, &prodNum, w, result)
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
		visitURL := webutil.getURL(prodName, pageNum)
		if prodNum <= maxProdNum {
			if err := c.Visit(visitURL); err != nil {
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
