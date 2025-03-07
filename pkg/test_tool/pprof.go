package testtool

import (
	"net/http"
	_ "net/http/pprof" // 匯入後會自動註冊 pprof endpoint
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/logger"
)

// StartPprof 根據環境變數啟動 pprof 監控伺服器
func StartPprof() {
	// 假設當環境變數 ENV 為 "production" 時，表示正式環境
	if !config.IsProduction() {
		logger.Log.Info("Production environment detected, pprof is disabled.")
		return
	}

	// 非 production 環境時，在預設 port 6060 上啟動 pprof 監控伺服器
	go func() {
		logger.Log.Info("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			logger.Log.Infof("pprof server failed: ", err)
		}
	}()
}

// pprof 會開啟一個 HTTP 伺服器，監聽 :6060 端口，提供以下分析端點：
// 	•	/debug/pprof/ → 顯示所有可用的分析數據
// 	•	/debug/pprof/goroutine → 顯示所有 Goroutines
// 	•	/debug/pprof/heap → 顯示記憶體分配
// 	•	/debug/pprof/profile → 執行 30 秒 CPU 分析
// 	•	/debug/pprof/block → 顯示 goroutine 阻塞的情況
// 	•	/debug/pprof/mutex → 顯示 mutex 鎖的競爭情況
// 	•	/debug/pprof/threadcreate → 顯示創建的系統執行緒數量

// (1) 確認 pprof 是否啟動
// ```
// curl http://localhost:6060/debug/pprof/
// ```
// 如果回傳 JSON，表示 pprof 正常運行。

// (2) 分析 CPU 使用情況
// 執行 30 秒 CPU Profile：
// ```
// go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
// ```
// (pprof) top 10    # 顯示 CPU 佔用最高的 10 個函數
// (pprof) list main.main  # 查看 main() 的詳細分析
// (pprof) web    # 生成可視化圖表 (需安裝 Graphviz)
// (pprof) quit   # 退出 pprof

// 最佳實踐：當你的應用程式 CPU 使用率高時，可以使用這個方法分析哪些函數佔用了最多 CPU 時間。

// (3) 分析記憶體使用情況
// ```
// go tool pprof http://localhost:6060/debug/pprof/heap
// ```
// (pprof) top 10    # 找出記憶體使用最多的函數
// (pprof) web    # 生成記憶體視覺化圖表

// 最佳實踐：當你的應用程式記憶體使用量高，或者出現 Memory Leak（記憶體洩漏） 時，這個方法可以幫助你找出問題。

// (4) 追蹤 Goroutine 狀態
// ```
// go tool pprof http://localhost:6060/debug/pprof/goroutine
// ```
// 這可以幫助你找出 哪個 goroutine 可能導致阻塞、資源爭用 等問題。

// (5) 下載並分析 Profiling 結果
// curl -o cpu.prof http://localhost:6060/debug/pprof/profile?seconds=30
// curl -o heap.prof http://localhost:6060/debug/pprof/heap

// go tool pprof cpu.prof
// go tool pprof heap.prof



// 雖然 pprof 很強大，但在使用時要注意以下問題：

// (1) pprof 會影響程式效能
// 在執行 pprof 時，程式會有額外的 Profiling Overhead（額外負擔），這可能會降低應用程式的效能。特別是：
// 	•	CPU Profiling 可能會影響效能（因為它需要額外計算函數執行時間）。
// 	•	Memory Profiling 可能會影響 Garbage Collector（GC）。
// 	•	解決方法：
// 	•	只在測試或開發環境啟用 ppro
// 	•	在 production 啟用時，請求 pprof 時避免過度頻繁，以免影響效能。

// (2) pprof 需要 HTTP 端口，可能帶來安全風險

// 目前你的 pprof 伺服器監聽 :6060，這個端口是開放的，這可能讓攻擊者可以存取 pprof 資料，獲取你的應用程式內部狀態。

// 解決方案：
// 	•	限制 pprof 的存取：在 main.go 中僅允許內部 IP 存取：
// ```
// if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
//     logger.Log.Errorf("pprof server failed: %v", err)
// }
// ```
// 這樣 pprof 只能在本機存取，而無法透過外部網路存取。

// •	使用 Basic Auth 保護 pprof：
// 你可以使用 http.HandlerFunc 來加上簡單的認證：
// ```
// func authMiddleware(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         user, pass, ok := r.BasicAuth()
//         if !ok || user != "admin" || pass != "password" {
//             w.WriteHeader(http.StatusUnauthorized)
//             return
//         }
//         next.ServeHTTP(w, r)
//     })
// }

// func StartPprof() {
//     if !config.IsProduction() {
//         return
//     }

//     mux := http.NewServeMux()
//     mux.Handle("/debug/pprof/", authMiddleware(http.DefaultServeMux))
    
//     go func() {
//         logger.Log.Info("Starting secured pprof server on :6060")
//         if err := http.ListenAndServe(":6060", mux); err != nil {
//             logger.Log.Errorf("pprof server failed: %v", err)
//         }
//     }()
// }
// ```
// 這樣 pprof 需要 帳號密碼驗證，提高安全性。

