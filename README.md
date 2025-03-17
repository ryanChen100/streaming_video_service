# 🎬 Streaming Video Service

**Streaming Video Service** 是一個 **可觀看串流影片並即時聊天** 的微服務架構專案，專為高併發與低延遲需求設計。本專案融合 **影音串流、即時通訊、身份驗證、分散式資料存儲**，並使用 **Docker 容器化** 以確保靈活部署與擴展性。

> **✨ 項目特色**
> - **微服務架構**：拆分 `api_gateway`、`chat_service`、`member_service`、`streaming_service` 等模組，各司其職
> - **高效通訊**：gRPC + Redis Pub/Sub 確保低延遲
> - **完整測試**：Mock Test + Integration Test 達成高可靠性
> - **高擴展性**：RabbitMQ 進行非同步處理，支持彈性擴展
> - **最佳開發實踐**：遵循 **DDD、BDD、TDD**，使用 **Gherkin** 進行行為驅動開發

---

## 🏛️ 架構設計

### 🔹 **系統架構圖**
```
#! chat_service 預計關閉對外端口

+--------------------+       +----------------------+  gRPC   +----------------+
| Client (Web/Mobile) | <--> | api_gateway          | <----> | Member Service  |
+--------------------+       +----------------------+         +----------------+
        |                             |gRPC
        v                             v
+------------------+          +---------------------+       +------------------+
| Chat Service     | <--> gRPC| Streaming Service   | <-->  | PostgreSQL       |
+------------------+          +---------------------+       +------------------+
        |           \                 
        v            \                
+-----------------+   +------------------+
| MongoDB (Msg)  |    | Redis (Pub/Sub) |
+-----------------+   +------------------+
```
### 🔹 **技術選型**
| 功能            | 技術 | 描述 |
|---------------|------|------|
| Web 框架 | **Fiber (Golang)** | 高效能的 Golang Web 框架 |
| API 通訊 | **gRPC + REST** | gRPC 提供內部微服務通訊，REST API 對外 |
| 身份驗證 | **JWT + Middleware** | 使用 JWT 進行 Token 驗證 |
| 即時通訊 | **Redis Pub/Sub** | 提供高效的即時消息傳遞 |
| 訊息儲存 | **MongoDB** | 以桶（Bucket）存儲聊天訊息 |
| 排程任務 | **RabbitMQ** | 處理非同步訊息與排程 |
| 測試 | **Mock Test + Integration Test** | 確保服務穩定性 |
| 服務部署 | **Docker + Docker Compose** | 容器化管理所有微服務 |

---

## 🚀 核心功能

### 🎥 **影音串流**
- 提供影片存取 API，支援高效能的 **分片存儲與載入**
- 可記錄 **觀看歷史**，推薦使用者感興趣的內容
- 支援使用者上傳影片，**並使用 FFmpeg 轉碼 成HLS**

### 💬 **即時聊天室**
- **Redis Pub/Sub** 進行即時通訊，減少輪詢開銷
- **MongoDB** 以 Bucket 模式儲存歷史訊息，提高查詢效能
- 使用 **WebSocket** 建立雙向即時連線

### 🔐 **身份驗證與 API 保護**
- **JWT Token 驗證**，確保用戶身份安全
- `api_gateway` **Middleware** 負責請求攔截
- **Swagger API 文檔** 提供開發者快速測試 API 介面

### 🏗 **微服務整合**
- 透過 **gRPC** 進行跨服務通訊，減少 HTTP 開銷
- **RabbitMQ** 進行異步任務處理，確保高吞吐量
- **Redis 哨兵模式（Sentinel）** 確保高可用性

---

## 📂 專案目錄結構
```
streaming_video_service/
│────cmd
│    │── api_gateway/         # API 入口點，身份驗證
│    │── chat_service/        # 即時聊天服務（使用 Redis Pub/Sub）
│    │── member_service/      # 使用者管理（JWT 驗證）
│    │── streaming_service/   # 影片管理與串流
│────internal
|    │── api_gateway/         # 實作API 入口點，身份驗證
│    │── chat_service/        # 實作即時聊天服務（使用 Redis Pub/Sub）
│    │── member_service/      # 實作使用者管理（JWT 驗證）
│    │── streaming_service/   # 實作影片管理與串流
│────pkg
│    │── database/            # 資料庫設定（MongoDB + Redis）
│── tests/                    # 測試（Mock Test & Integration Test）
│── docker-compose.yml        # 容器化設定
└── README.md                 # 本文件

```
## 🔥 **開發與測試**
本專案嚴格遵循 **BDD（行為驅動開發）**，並搭配 **TDD（測試驅動開發）**，確保代碼穩定性與可測試性。

### ✅ **測試執行**
執行所有測試：
```sh
go test ./...

	•	Mock Test：針對每個 service 撰寫 Mock 測試
	•	Integration Test：確保 api_gateway 和 microservices 能夠正確互動
	•	行為測試（BDD）：使用 Gherkin 編寫測試場景

```

## ⚡ 快速啟動專案

1️⃣ 啟動專案

確保已安裝 Docker，然後執行：
```
docker-compose up -d
```

2️⃣ 確認服務運行
檢查容器：
```
docker ps
```

3️⃣ 透過 Swagger 瀏覽 API
[swagger](http://localhost:8080/swagger)

🌟 為何選擇這些技術？

	1️⃣ 選擇 Fiber 的原因？
		•	高效能：相比於標準 net/http，Fiber 使用 fasthttp，在處理 高並發請求 時有更好的效能
	•	簡潔 API：使用類似 Express.js 的風格，開發體驗優秀

	2️⃣ gRPC 的優勢？
		•	二進制協議：相比 REST API，gRPC 使用 Protocol Buffers，能顯著降低 數據傳輸量 和 序列化開銷
	•	內建流式傳輸：更適合處理即時串流、多人聊天室等需求

	3️⃣ 為何 Redis Pub/Sub？
		•	極低延遲：適合即時通訊場景
	•	簡單輕量：不需要額外維護長連接管理

	4️⃣ MongoDB 用桶（Bucket）來存訊息的原因？
		•	高效批量查詢：將 多條訊息打包存儲，相比單條存儲，查詢效率更高
	•	靈活擴展性：MongoDB 天然適合儲存非結構化數據

## 📌 未來計畫
### ✅ 短期目標（近期內可實現）
	•	🔜 chat_service 統一使用 api_gateway 入口點，避免對外暴露不必要的端口，提高安全性
	•	🔜 完善聊天室訊息歷史查詢，提供分頁與搜尋功能，優化 MongoDB 索引，提高查詢效能
	•	🔜 新增 FFmpeg 轉碼，支援 DASH 影片格式，讓影片串流更加靈活，適應不同裝置與網路環境
	•	🔜 API Gateway 增加流量管理與限流，使用 Rate Limiting 防止 DDoS 攻擊，提高 API 穩定性
	•	🔜 引入 Kubernetes，提升微服務擴展性，確保系統具備自動擴容能力

### 🚀 中期目標（技術升級與優化）
	•	🔜 引入 OpenTelemetry 進行全鏈路追蹤（Tracing），監控 gRPC、Redis、RabbitMQ 等核心通訊，方便故障排查
	•	🔜 WebRTC 支援，未來讓直播功能更流暢，減少伺服器負擔
	•	🔜 改進 WebSocket 連線管理，透過 Redis Cluster 分片，提高併發量
	•	🔜 MongoDB 資料分片（Sharding），提升歷史訊息的查詢效能
	•	🔜 Redis 換成 Cluster 模式，增強高可用性與負載均衡能力
	•	🔜 RabbitMQ 換成 Kafka，用於更大規模的訊息處理，確保可擴展性
	•	🔜 多語言支援，為聊天訊息與影片字幕加入 AI 自動翻譯功能
	•	🔜 增加會員等級與訂閱機制，讓 VIP 會員享受更高品質的串流服務
	•	🔜 導入 AI 影片推薦系統，根據觀看行為推薦個人化內容

### 🌍 長期目標（戰略性擴展）
	
	•	🔜 透過 CDN 加速影片傳輸，減少後端伺服器負載
	•	🔜 上架到雲端平台（AWS/GCP/Azure），確保系統彈性與可擴展性
	•	🔜 新增 Web3 相關功能，如 NFT 影片存證，提供影片創作者更多獲利機會
	•	🔜 支援 Serverless 架構，部分 API 服務可無伺服器運行，降低基礎設施成本
	•	🔜 支援 P2P 影片分發技術，減少伺服器壓力，降低頻寬成本
	•	🔜 自動化 DevOps，CI/CD 優化，整合 GitHub Actions 或 ArgoCD，提高部署效率
	

🏆 結語

此專案不僅是一個影音平台的後端架構示範，更是一個高效能、可擴展的 微服務架構。透過 gRPC、Redis、RabbitMQ 這些技術，打造高併發低延遲的體驗。希望這個專案能展示我的技術選型與系統設計能力！