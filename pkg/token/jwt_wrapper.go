package token

import "streaming_video_service/pkg/config"

// 這個變數會在測試時被覆蓋
var (
	GenerateJWTFunc = GenerateJWT
	ParseJWTFunc    = ParseJWT
)

// 使用這種**包裝函數（wrapper functions）**來取代標準函數調用，對效能的影響可以從幾個方面來考慮：

// 1. 影響效能的因素

// 這種包裝函數的主要目的是為了可測試性（testability），而非性能優化，因此它可能會有一些微小的開銷，但在大多數應用中，這些開銷是可以忽略的。
// 	•	函數調用的開銷
// 	•	這些變數（createDir、createFile 等）實際上是函數變數（function variables），Go 會在執行時透過間接調用（indirect function call）來執行它們，這比直接調用 os.MkdirAll 或 os.Create 稍微 慢一點。
// 	•	但這種額外的開銷 通常不會對整體效能造成明顯影響，除非這些函數被高頻率地調用，例如每毫秒調用成千上萬次。
// 	•	函數指標的 CPU 分支預測（Branch Prediction）影響
// 	•	Go 直接調用內建函數時，編譯器可以優化函數內聯（Inlining），減少 CPU 分支錯誤的機會。
// 	•	但當你使用函數變數時，Go 必須在執行時解析這個變數，這可能導致 CPU 分支預測（Branch Prediction Miss），影響 CPU pipeline，但這種影響通常微乎其微。
// 	•	垃圾回收（GC）影響
// 	•	這種函數變數通常不會造成逃逸（escape to heap），因為它們本質上只是函數指標，存放在 Go 的 data segment（全局變數區域）。
// 	•	但如果在測試時頻繁改變這些函數，可能會有額外的 GC 負擔（不過影響很小）。

// 2. 這樣的設計優勢
// 	•	可測試性
// 	•	你可以在測試時替換這些函數，模擬 I/O 失敗，例如 createFile = func(name string) (*os.File, error) { return nil, errors.New("mock error") }。
// 	•	減少了對真實文件系統的依賴，讓測試更快、更穩定。
// 	•	更靈活的控制
// 	•	可以在不同的環境（如本地測試 vs. 生產環境）中，使用不同的實現，例如：
//! 在高併發的場景下 （如 每秒成千上萬次呼叫） 還是內關聯具有優勢
// ```
// if isTesting {
//     createFile = mockCreateFile
// } else {
//     createFile = os.Create
// }
// ```
// 3. 什麼情況下可能影響效能？
// 	1.	高頻率調用
// 	•	如果 createFile()、copyFile() 被頻繁調用（如每個請求執行數千次 I/O 操作），這可能會帶來可測量的效能影響。
// 	•	解決方案： 在性能關鍵的地方，直接調用標準函數 os.Create()、io.Copy()，避免間接調用。
// 	2.	內聯優化無法進行
// 	•	Go 的編譯器對內聯（Inlining）有一定的限制，而函數變數會阻止這種優化。
// 	•	例如 readFile = func(r io.Reader) ([]byte, error) { return io.ReadAll(r) } 這樣的簡單函數，Go 原本可能會內聯 io.ReadAll(r)，但現在無法內聯，影響 CPU 指令調度的效率。
// 	•	解決方案： 對於確定不需要 mock 的函數，直接使用標準函數，避免這種不必要的包裝。
// 	3.	函數變數的動態變更
// 	•	如果程式碼頻繁修改 createFile 或 readFile，這會影響 Go 內部的優化機制，造成額外的記憶體開銷。
// 	•	解決方案： 減少變更這些函數變數的次數，並在測試時只修改一次。

// 4. 結論

// ✅ 影響很小，大多數情況下可忽略
// 	•	普通應用場景（如 API 服務）這種設計的影響微乎其微，且帶來了更好的測試能力。
// 	•	但在高效能場景（如高並發 I/O 操作），直接使用標準函數可能會更好。

// 📌 最佳實踐
// 	1.	對需要 mock 的函數使用包裝函數（提高測試性）。
// 	2.	性能關鍵部分避免包裝函數，直接使用標準函數。
// 	3.	不要頻繁變更這些函數變數，避免影響 Go 內部優化。

// GenerateJWTWrapper 讓 `memberUseCase` test mock使用這個包裝函數
func GenerateJWTWrapper(memberID, role string) (string, error) {
	return GenerateJWTFunc(memberID, role, config.EnvConfig.MemberService)
}

// ParseJWTWrapper 讓 `memberUseCase`  test mock使用這個包裝函數
func ParseJWTWrapper(t string) (*Claims, error) {
	return ParseJWTFunc(t)
}
