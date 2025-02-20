package handlers

// import (
// 	"context"
// 	"strconv"
// 	"time"

// 	"streaming_video_service/internal/chat/domain"
// 	"streaming_video_service/pkg/logger"
// 	"streaming_video_service/pkg/middlewares"
// 	chatpb "streaming_video_service/pkg/proto/chat"

// 	"github.com/gofiber/websocket/v2"
// 	"go.uber.org/zap"
// )

// // ChatWebsocketHandler 用於處理 WebSocket 聊天連線，並透過 gRPC 呼叫 chat 服務
// type ChatWebsocketHandler struct {
// 	ChatClient chatpb.ChatServiceClient
// }

// // NewChatWebsocketHandler 建構 ChatWebsocketHandler
// func NewChatWebsocketHandler(client chatpb.ChatServiceClient) *ChatWebsocketHandler {
// 	return &ChatWebsocketHandler{
// 		ChatClient: client,
// 	}
// }

// // HandleConnection 處理 WebSocket 連線
// func (h *ChatWebsocketHandler) HandleConnection(ctx context.Context, conn *websocket.Conn) {
// 	tokenMember := conn.Locals(middlewares.TokenMemberID)
// 	memberID, ok := tokenMember.(string)
// 	logger.Log.Info("websocket handle memberID", zap.String("userID", memberID), zap.String("ok", strconv.FormatBool(ok)))

// 	ticker := time.NewTicker(10 * time.Minute)
// 	ctxClose, cancel := context.WithCancel(context.Background())

// 	defer func() {
// 		ticker.Stop()
// 		logger.Log.Info("websocket close", zap.String("userID", memberID))
// 		conn.Close()
// 		cancel()
// 	}()

// 	//client發出close
// 	//fiber會自動處理(在read msg 回傳err),故需要SetCloseHandler另外接出
// 	conn.SetCloseHandler(func(code int, text string) error {
// 		logger.Log.Infof("WebSocket closed:", conn.RemoteAddr())
// 		return nil
// 	})

// 	//server發出ping之後client連線正常會回pong
// 	//fiber會自動處理回傳pong,故需要SetPongHandler另外接出
// 	conn.SetPongHandler(func(appData string) error {
// 		logger.Log.Infof("Received PONG:", appData)
// 		return nil
// 	})

// 	//client發出ping
// 	//fiber會自動處理ping,故需要SetPingHandler另外接出
// 	conn.SetPingHandler(func(appData string) error {
// 		logger.Log.Infof("Received PING:", appData)
// 		// 如果要手動回 Pong，可以：
// 		return conn.WriteControl(
// 			websocket.PongMessage,
// 			[]byte(appData), // 一般可帶原封不動的資料
// 			time.Now().Add(time.Second),
// 		)
// 	})

// 	//啟用sub訂閱自己的訊息
// 	channel := "chat:user:" + memberID
// 	h.messageUC.memberPubSub.Subscribe(ctxClose, channel, func(resp domain.WSResponse) {
// 		h.sendResponse(conn, resp)
// 	})

// 	// 定期發送 Ping
// 	go func() {
// 		for {
// 			select {
// 			case <-ticker.C:
// 				// 發送 Ping 消息
// 				pingMsg := "ping message"
// 				if err := conn.WriteMessage(websocket.PingMessage, []byte(pingMsg)); err != nil {
// 					logger.Log.Errorf("Ping error:", err)
// 					return
// 				}
// 				logger.Log.Infof("%s Ping sent", memberID)
// 			case <-ctxClose.Done():
// 				logger.Log.Infof("Ping goroutine cancelled for member:", memberID)
// 				return
// 			}
// 		}
// 	}()

// 	for {
// 		// 1. 讀取前端訊息
// 		mt, message, err := conn.ReadMessage()
// 		if err != nil {
// 			// 檢查是否為 Close 正常結束
// 			if websocket.IsCloseError(err,
// 				websocket.CloseNormalClosure,
// 				websocket.CloseGoingAway,
// 				websocket.CloseNoStatusReceived, //1005 c.WriteMessage(websocket.CloseMessage, []byte{})
// 			) {
// 				logger.Log.Errorf("Connection closed:", err)
// 			} else {
// 				//直接斷線 1006
// 				logger.Log.Errorf("websocket read error:", err)
// 			}
// 			return
// 		}
// 		h.execWebsocketAction(ctx, conn, memberID, mt, message)
// 	}
// }
