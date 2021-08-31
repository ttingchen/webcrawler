# Golang-WebCrawler
## Description

請設計一個 Web Service，輸入關鍵字後，搜尋購物網站並把結果呈現出來

## 需求
* **RESTful API**
* **善用 interface 抽換底層實作，讓 code 具有延展性，並更容易測試**
* **Use context to replace flag shutdown**
* 至少兩個購物網站
* 商品資訊至少包含：名稱、價錢、圖片連結、商品連結
* Exported functions / variables 要有註解
* 要寫 Unit test
* 運用 worker 技巧，並提供 flag 設定單一網站最多 workers 數量
* 搜尋結果有多頁時，也要爬下來
* 程式被中斷時，worker 必須把手上任務完成才結束

## 加分
* 若等每一頁的結果都收集好才回傳，User 可能會等很久，UX 不佳；
  請嘗試 real time render
* 運用 Database 建立 cache 機制，特定期限內 user 再次搜尋相同關鍵字，就不用再爬一次
  但請勿 hard-code DB 連線資訊
* 結構化的 log 有助於了解程式運作情形，有效 debug

## 提示
* 善用 opensource libraries
* 避免頻繁的 request 導致購物網站誤判 DDoS 攻擊，請注意 rate limit
* 避免商品太多，可設定數量上限，或是過濾缺貨商品
