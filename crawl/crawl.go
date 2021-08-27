package crawl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

// Product is product
type Product struct {
	Name  string `json:"Name"`
	Price string `json:"Price"`
	Image string `json:"Image"`
	URL   string `json:"URL"`
}

type webUtil interface {
	onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, result *[]Product) error
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

// max product amount for each online store
const (
	maxProdNum = 100
)

func searchWeb(ctx context.Context, prodName string, w http.ResponseWriter, r *http.Request) (*[]Product, error) {
	var ebayInfo webUtil = &ebayUtil{
		Name:       "Ebay",
		NumPerPage: 50,
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
	var mu sync.Mutex
	c := make(chan error, 2)

	for _, website := range websites {

		go func(web webUtil) {
			crawlWebsite(ctx, c, &mu, web, prodName, &result, w, r)
		}(website)
	}
	for err := range c {
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("done err waiting")
	return &result, nil
}

// scrape product info from website
func crawlWebsite(rctx context.Context, errchan chan error, mu *sync.Mutex, webutil webUtil, prodName string, result *[]Product, w http.ResponseWriter, r *http.Request) {
	Err := ""
	webinfo := webutil.getInfo()
	wg := sync.WaitGroup{}

	maxPageNum := (maxProdNum / 2) / webinfo.NumPerPage
	fmt.Println("new collector: ", webinfo.Name)
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent(webinfo.UserAgent),
	)

	collyctx := colly.NewContext()    // 建立新的 colly.Context
	collyctx.Put("request_ctx", rctx) // 把 request context 放進 colly

	c.Limit(&colly.LimitRule{
		// Set a delay between requests to these domains
		Delay: 3 * time.Second,
		// Add an additional random delay
		RandomDelay: 15 * time.Second,

		Parallelism: 3,
	})

	c.OnHTML(webinfo.OnHTML, func(e *colly.HTMLElement) {
		// for each website
		webutil.onHTMLFunc(e, mu, w, result)
	})

	c.OnError(func(r *colly.Response, err error) {
		Err = fmt.Sprintln("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnRequest(func(r *colly.Request) {
		v := r.Ctx.GetAny("request_ctx")
		ctx, ok := v.(context.Context)
		if !ok {
			fmt.Println("context type error")
			r.Abort()
			return
		}
		select {
		case <-ctx.Done(): // 如果 canceled
			fmt.Println("context done")
			r.Abort() // 結束 request
			Err = fmt.Sprintln("context done")
		default: // 要有 default，不然 select {} 會卡住
		}
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("On Scraped, wait group done")
		wg.Done()
	})

	//load 1 to pageNum pages
	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {
		visitURL := webutil.getURL(prodName, pageNum)
		wg.Add(1)
		if err := c.Request(http.MethodGet, visitURL, nil, collyctx, nil); err != nil {
			log.Println("Url err:", err)
		}
	}

	wg.Wait()
	fmt.Println("Done waiting")

	if Err != "" {
		errchan <- errors.New(Err)
		close(errchan)
	}
	errchan <- nil

}

func (u *ebayUtil) onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, result *[]Product) (err error) {
	buf := new(bytes.Buffer)
	if len(*result) < maxProdNum {
		//avoid to get a null item
		if e.ChildText("h3[class='s-item__title']") != "" {
			//use regex to remove the useless part of prodlink
			re := regexp.MustCompile(`\?(.*)`)
			prodName := e.ChildText("h3[class='s-item__title']")
			prodLink := e.ChildAttr("a[class='s-item__link']", "href")
			prodLinkR := re.ReplaceAllString(prodLink, "")
			prodImgLink := e.ChildAttr("img[class='s-item__image-img']", "src")
			prodPrice := e.ChildText("span[class='s-item__price']")

			prod := Product{prodName, prodPrice, prodImgLink, prodLinkR}

			*result = append(*result, prod)
			if err = json.NewEncoder(buf).Encode(prod); err != nil {
				fmt.Println(err)
				return
			}
			m.Lock()
			fmt.Fprintf(w, "Ebay #%v: json.NewEncode:\n", len(*result))
			io.Copy(w, buf)
			fmt.Fprintf(w, "\n")
			m.Unlock()
		}
	}
	return err
}

func (u *ebayUtil) getURL(prodName string, pageNum int) string {
	return "https://www.ebay.com/sch/i.html?_nkw=" + prodName + "&_ipg=50&_pgn=" + strconv.Itoa(pageNum)
}

func (u *ebayUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
		UserAgent:  u.UserAgent,
	}
}

func (u *watsonsUtil) onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, result *[]Product) (err error) {
	buf := new(bytes.Buffer)
	e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
		time.Sleep(100 * time.Millisecond) // to observe the goroutine
		prodName := e.ChildText(".productName")
		prodLink := "https://www.watsons.com.tw" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href")
		prodImgLink := e.ChildAttr("img", "src")
		prodPrice := e.ChildText(".productPrice")
		prod := Product{prodName, prodPrice, prodImgLink, prodLink}

		*result = append(*result, prod)
		n := len(*result)
		if err = json.NewEncoder(buf).Encode(prod); err != nil {
			fmt.Println(err)
			return
		}
		m.Lock()
		fmt.Fprintf(w, "Watsons #%v: json.NewEncode:\n", n)
		io.Copy(w, buf)
		fmt.Fprintf(w, "\n")
		m.Unlock()

	})

	return err
}

func (u *watsonsUtil) getURL(prodName string, pageNum int) string {
	return fmt.Sprintf("https://www.watsons.com.tw/search?text=%v&useDefaultSearch=false&pageSize=64&currentPage=%d", prodName, pageNum-1)
}

func (u *watsonsUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
		UserAgent:  u.UserAgent,
	}
}
