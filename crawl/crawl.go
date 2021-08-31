package crawl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

// Total max product amount.
const maxProdNum = 500

// Product is the struct of product information including name, price, imagelink, url.
type Product struct {
	Name  string `json:"Name"`
	Price string `json:"Price"`
	Image string `json:"Image"`
	URL   string `json:"URL"`
}

type watsonsUtil webInfo
type ebayUtil webInfo

// SearchWeb uses a colly collector to crawl for each website.
func SearchWeb(ctx context.Context, prodName string, w http.ResponseWriter, r *http.Request) (*[]string, error) {
	var ebayInfo webUtil = &ebayUtil{
		Name:       "Ebay",
		NumPerPage: 50,
		OnHTML:     "div[class='s-item__wrapper clearfix']",
		Parallel:   13,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36",
	}
	var watsonInfo webUtil = &watsonsUtil{
		Name:       "Watsons",
		NumPerPage: 64,
		OnHTML:     "e2-product-list",
		Parallel:   3,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36",
	}

	websites := []webUtil{
		ebayInfo,
		watsonInfo,
	}

	var resultJSON []string
	var mu sync.Mutex
	var Err error
	ch := make(chan error, 2)
	counter := 0

	for _, website := range websites {
		go func(web webUtil) {
			crawlWebsite(ctx, ch, &mu, web, prodName, &resultJSON, w)
		}(website)
	}

	for err := range ch {
		counter++
		if err != nil {
			Err = err
		}
		if counter == len(websites) {
			break
		}
	}
	fmt.Println("Done err waiting")
	return &resultJSON, Err
}

// LogResults logs the scraped results.
func LogResults(ctx context.Context, searchResult *[]string) error {
	fmt.Println("Start to log results")
	for i, result := range *searchResult {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		var product Product
		if err := json.NewDecoder(strings.NewReader(result)).Decode(&product); err != nil {
			return err
		}
		fmt.Printf("Total #%d : \n%v\n%v\n%v\n%v\n\n", i+1, product.Name, product.URL, product.Image, product.Price)

	}
	return nil
}

func crawlWebsite(rctx context.Context, errchan chan error, mu *sync.Mutex, webutil webUtil, prodName string, resultJSON *[]string, w http.ResponseWriter) {
	var Err error
	webinfo := webutil.getInfo()
	wg := sync.WaitGroup{}

	maxPageNum := (maxProdNum / 2) / webinfo.NumPerPage
	fmt.Println("new collector:", webinfo.Name)
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent(webinfo.UserAgent),
	)

	// create a new colly.Context
	collyctx := colly.NewContext()
	// put request context into colly
	collyctx.Put("request_ctx", rctx)

	c.Limit(&colly.LimitRule{
		Delay:       3 * time.Second,  // set a delay between requests to these domains
		RandomDelay: 15 * time.Second, // add an additional random delay
		Parallelism: webinfo.Parallel,
	})

	c.OnHTML(webinfo.OnHTML, func(e *colly.HTMLElement) {
		// implented different interface for each website
		if err := webutil.onHTMLFunc(e, mu, w, resultJSON); err != nil {
			Err = errors.New(fmt.Sprintln(Err, err))
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("On error")
		Err = errors.New(fmt.Sprintln(Err, err))
		wg.Done()
	})

	c.OnScraped(func(r *colly.Response) {
		v := r.Ctx.GetAny("request_ctx")
		ctx, ok := v.(context.Context)
		if !ok {
			fmt.Println("Context type error")
			return
		}
		select {
		case <-ctx.Done():
			fmt.Println("Context done")
			Err = context.Canceled
		default:
		}

		wg.Done()
	})

	for pageNum := 1; pageNum <= maxPageNum; pageNum++ {
		visitURL := webutil.getURL(prodName, pageNum)
		wg.Add(1)
		if err := c.Request(http.MethodGet, visitURL, nil, collyctx, nil); err != nil {
			log.Println("Url err:", err)
		}
	}

	wg.Wait()
	fmt.Println("Done waiting")

	if Err != nil {
		errchan <- Err
		return
	}
	errchan <- nil
	return
}
