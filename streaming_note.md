流程
	1.	上傳 API
	•	使用 Fiber 接收影片上傳請求
	•	將上傳的檔案先暫存，再上傳到 MinIO（物件儲存）
	•	將影片相關 Metadata（例如 title、description、檔案在 MinIO 的 object key、影片類型、狀態等）寫入 PostgreSQL
	2.	MinIO 客戶端
	•	包含上傳（UploadFile）與下載（DownloadFile）功能
	•	連線時帶有重試機制（例如 NewMinIOConnection 中嘗試重連）
	3.	FFmpeg 轉碼模組
	•	提供將影片轉碼成 HLS 與 DASH 格式的函式
	•	這部分在 Worker 或消費端可以使用
	4.	Worker 模組
	•	掃描資料庫中狀態為 “uploaded” 的影片，並透過 RabbitMQ 與 Kafka 將轉碼工作訊息發送出去
	•	更新影片狀態為 “processing”
	•	雖然這裡只示範了發送工作訊息，但消費端的實作則負責下載、轉碼、上傳處理結果並更新狀態為 “ready”
	5.	資料庫存取層 (repository)
	•	定義了 Video 模型，並提供基本的 CRUD、搜尋（利用 ILIKE）與簡單推薦（根據 view_count 排序）

知識補充
建立一個完整串流平台需要具備的知識
要打造一個真正完整的串流平台，除了上述功能，還需要掌握以下幾個領域的知識：
	1.	物件儲存與 CDN
	•	了解 MinIO / S3 的基本概念（Bucket、Object、S3 API）
	•	CDN 的運作與緩存原理，如何配置 CDN 將影片分發給全球用戶
	2.	資料庫管理與最佳化
	•	PostgreSQL 的使用、性能調優、索引設計
	•	資料庫連線池與重試機制（如我們之前提到的 retry 機制）
	3.	影片處理與轉碼技術
	•	FFmpeg 的使用、轉碼參數、如何產生 HLS/DASH 的多碼率串流
	•	理解串流格式：
	•	HLS (HTTP Live Streaming)：由 Apple 推出，利用 m3u8 播放清單與 TS 分段，適合 iOS/macOS 並廣泛支援瀏覽器。
	•	DASH (Dynamic Adaptive Streaming over HTTP)：國際標準，使用 MPD 播放清單；兩者的原理類似，主要差異在格式與部分參數上。
	4.	消息佇列與分散式處理
	•	了解 RabbitMQ 與 Kafka 的基本原理、使用方式與佈局
	•	消息的發布/訂閱模式、重試與斷線重連機制
	•	如何設計 Worker 來進行分散式轉碼（削峰）
	5.	API 與後端架構
	•	RESTful API 設計，如何使用 Fiber（或其他 Go 框架）建立 API 服務
	•	服務的容錯、重試、監控（日誌、metrics）
	6.	容器化與微服務部署
	•	Docker 與 Docker Compose 的使用，如何管理多個服務（如 PostgreSQL、MinIO、RabbitMQ、Kafka）
	•	未來如果要做大規模部署，可以進一步學習 Kubernetes 等容器編排工具
	7.	安全性與用戶認證
	•	雖然你已有 member_service，但在整合時也要考慮 API 的授權、檔案存取控制、HTTPS 等安全性議題
	8.	前端播放技術
	•	HLS.js、Dash.js 的使用，如何在網頁上播放串流影片
	•	可能還需了解自適應串流的原理


	1.	上傳 API (Producer 部分)
當使用者透過 API 上傳影片時：
	•	你的上傳 API 接收影片並暫存到本地，然後利用 MinIO 客戶端將影片上傳到 MinIO 的某個 bucket（例如 “original” 目錄下）。
	•	同時，上傳 API 會將影片的 metadata（例如 title、description、MinIO 中的 object key、影片類型等）存入 PostgreSQL，並將影片狀態標記為 “uploaded”。
	•	上傳 API 還會發送一個轉碼工作訊息到消息佇列（RabbitMQ 或 Kafka），這就是 Producer 的部分。
	2.	轉碼處理 (Consumer 部分)
	•	另一個獨立運作的 Consumer（或 Worker 消費者）會持續監聽消息佇列，當有 Producer 發送的轉碼訊息時，Consumer 就會接收到訊息。
	•	消費端根據訊息內容：
	•	從 MinIO 下載該影片（使用 DownloadFile 方法）到本地暫存區。
	•	使用 FFmpeg 轉碼該影片成 HLS 格式（也可以選擇轉 DASH 格式或實現多碼率，依需求調整）。
	•	將轉碼完成的 HLS 檔案（包含 index.m3u8 與各個 TS 分段檔）上傳到 MinIO 的另一個位置（例如 “processed/{videoID}/”）。
	•	更新 PostgreSQL 中該影片的狀態為 “ready”，表示影片已完成轉碼並可供播放。
	•	清理本地的暫存檔案。
	3.	播放影片
當使用者想播放影片時：
	•	前端會根據影片 title（或其他關鍵字）向後端搜尋該影片，後端會從 PostgreSQL 中讀取影片 metadata，包括存放轉碼後 HLS 檔案的路徑（例如 “processed/{videoID}/index.m3u8”）。
	•	前端獲得播放 URL 後，再利用 HLS 播放器（如 HLS.js、或是 iOS 原生播放器）播放影片。

總結來說：
	•	上傳 API 負責接收影片、存入 MinIO 與 PostgreSQL，並發送轉碼訊息（Producer）。
	•	獨立的 Consumer 監聽消息佇列，一旦收到訊息就進行影片轉碼（下載原始影片 → FFmpeg 轉碼 → 上傳轉碼結果 → 更新資料庫）。
	•	播放時透過搜尋影片 title 取得 HLS 播放 URL，前端使用 HLS 播放器播放影片。

這個流程基本上就構成了一個 VOD / Shorts 平台的核心，未來你可以根據需求加入更多功能，例如多碼率、CDN 整合、搜尋推薦、安全認證等等。