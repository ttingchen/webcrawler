# Golang-WebCrawler
## 簡介
設計一個 Web Service，能依據關鍵字對兩個購物網站 (Waston, Ebay )進行爬蟲，並將結果呈現出來

## 基本架構
* 利用 HTTP Handler 建構一個基礎的 Web API
* 利用第三方爬蟲框架 Colly 來實現爬蟲的基本需求
* 利用 JSON 來儲存爬蟲結果，提供了資料的高相容性以及未來擴展開發的便利性

## 其他細節
* 爬下來的商品資訊包含：名稱、價錢、圖片連結、商品連結
* 運用 Colly 的 Parallelism, Async 參數來實現 worker
* 運用 Colly 的 Limit, UserAgent 等參數來模擬真實使用者狀態來避免被網站封鎖
* 利用 interface 抽換底層實作，讓 code 具有延展性，並更容易測試
* 利用 context 來實現 graceful shutdown
* 搜尋結果有多頁時，可自動依據所設定商品數量計算頁數爬蟲
* 程式被中斷時，worker 能先將手上任務完成才結束
* 使用 mutex 來避免 HTTP Writer 造成的 race condition
* 基於 HTTP Writer 和 Colly 的並用來實現 real time render

## 尚可改進目標
* 運用 Database 建立 cache 機制，特定期限內 user 再次搜尋相同關鍵字，就不用再爬一次，但應避免 hard-code DB 連線資訊

## 使用方式
* 打開本地端任一瀏覽器於網址輸入 
  >localhost:`port number`/search?keyword=`your keyword`
  
  <img width="581" alt="search_chrome" src="https://user-images.githubusercontent.com/10221555/131460216-10fcbda8-66f0-4ad9-ad8e-adb0d1096d51.png">


* 或是打開終端機 
  > curl 'localhost:`port number`/search?keyword=`your keyword`'
  
  ![search_cmd](https://user-images.githubusercontent.com/10221555/131459999-51f7a9b0-4a79-41cc-a5dc-e9b593bdd02c.png)

## 爬蟲結果
<img width="1438" alt="search_result" src="https://user-images.githubusercontent.com/10221555/131463539-014f85ac-a046-4762-ae01-9761d1adcbe4.png">
