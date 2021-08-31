package crawl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

type webUtil interface {
	onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, resultJSON *[]string) error
	getURL(prodName string, pageNum int) string
	getInfo() webInfo
}

type webInfo struct {
	Name       string
	NumPerPage int
	OnHTML     string
	Parallel   int
	UserAgent  string
}

func (u *ebayUtil) onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, resultJSON *[]string) (err error) {
	if len(*resultJSON) < maxProdNum {
		//avoid to get a null item
		if e.ChildText("h3[class='s-item__title']") != "" {
			//use regex to remove the useless part of prodlink
			re := regexp.MustCompile(`\?(.*)`)
			prodName := e.ChildText("h3[class='s-item__title']")
			prodLink := e.ChildAttr("a[class='s-item__link']", "href")
			prodLinkR := "<a href=\"" + re.ReplaceAllString(prodLink, "") + "\">click this</a>"
			prodImgLink := "<img src=\"" + e.ChildAttr("img[class='s-item__image-img']", "src") + "\" width=\"400\" height=\"500\">"
			prodPrice := e.ChildText("span[class='s-item__price']")

			m.Lock()
			fmt.Fprintf(w, "Ebay #%v:<br>", len(*resultJSON)+1)
			fmt.Fprintf(w, "Name: %v<br>", prodName)
			fmt.Fprintf(w, "ProdLink: %v<br>", prodLinkR)
			fmt.Fprintf(w, "ImageLink: %v<br>", prodImgLink)
			fmt.Fprintf(w, "Price: %v<br><br>", prodPrice)

			prod := Product{prodName, prodPrice, prodImgLink, prodLinkR}
			buf := new(bytes.Buffer)
			if err = json.NewEncoder(buf).Encode(prod); err != nil {
				fmt.Println(err)
				return
			}
			str := string(buf.Bytes())
			*resultJSON = append(*resultJSON, str)
			// fmt.Fprintf(w, "Ebay #%v: json.NewEncode:\n", len(*resultJSON))
			// io.Copy(w, buf)
			// fmt.Fprintf(w, "\n")
			m.Unlock()
		}
	}
	return err
}

func (u *ebayUtil) getURL(prodName string, pageNum int) string {
	return fmt.Sprintf("https://www.ebay.com/sch/i.html?_nkw=%v&_ipg=50&_pgn=%d", prodName, pageNum)
}

func (u *ebayUtil) getInfo() webInfo {
	return webInfo{
		Name:       u.Name,
		NumPerPage: u.NumPerPage,
		OnHTML:     u.OnHTML,
		Parallel:   u.Parallel,
		UserAgent:  u.UserAgent,
	}
}

func (u *watsonsUtil) onHTMLFunc(e *colly.HTMLElement, m *sync.Mutex, w http.ResponseWriter, resultJSON *[]string) (err error) {
	e.ForEach("e2-product-tile", func(_ int, e *colly.HTMLElement) {
		// add sleep() to observe the goroutine
		time.Sleep(100 * time.Millisecond)
		prodName := e.ChildText(".productName")
		prodLink := "<a href=\"https://www.watsons.com.tw\"" + e.ChildAttr(".ClickSearchResultEvent_Class.gtmAlink", "href") + "\">click this</a>"
		prodImgLink := "<img src=\"" + e.ChildAttr("img", "src") + "\" width=\"400\" height=\"500\">"
		prodPrice := e.ChildText(".productPrice")

		m.Lock()
		fmt.Fprintf(w, "Watsons #%v:<br>", len(*resultJSON)+1)
		fmt.Fprintf(w, "Name: %v<br>", prodName)
		fmt.Fprintf(w, "ProdLink: %v<br>", prodLink)
		fmt.Fprintf(w, "ImageLink: %v<br>", prodImgLink)
		fmt.Fprintf(w, "Price: %v<br><br>", prodPrice)

		prod := Product{prodName, prodPrice, prodImgLink, prodLink}
		buf := new(bytes.Buffer)
		if err = json.NewEncoder(buf).Encode(prod); err != nil {
			fmt.Println(err)
			return
		}
		str := string(buf.Bytes())
		*resultJSON = append(*resultJSON, str)
		// fmt.Fprintf(w, "Watsons #%v: json.NewEncode:\n", len(*resultJSON))
		// io.Copy(w, buf)
		// fmt.Fprintf(w, "\n")
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
		Parallel:   u.Parallel,
		UserAgent:  u.UserAgent,
	}
}
