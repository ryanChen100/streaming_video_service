package app

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"streaming_video_service/internal/chat/domain"
	"streaming_video_service/pkg/logger"
	"streaming_video_service/pkg/middlewares"
	memberpb "streaming_video_service/pkg/proto/member"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

// ChatWebsocketHandler 可包含所有需要的 UseCase
type ChatWebsocketHandler struct {
	roomUC       *RoomUseCase
	messageUC    *SendMessageUseCase
	memberClient *memberpb.MemberServiceClient
	roomCtx      context.CancelFunc
}

// NewChatWebsocketHandler create ChatWebsocketHandler
func NewChatWebsocketHandler(
	roomUC *RoomUseCase,
	messageUC *SendMessageUseCase,
	memberUc *memberpb.MemberServiceClient,
) *ChatWebsocketHandler {
	return &ChatWebsocketHandler{
		roomUC:       roomUC,
		messageUC:    messageUC,
		memberClient: memberUc,
	}
}

// HandleConnection 是 WebSocket 連線的進入點
func (h *ChatWebsocketHandler) HandleConnection(ctx context.Context, conn *websocket.Conn) {
	tokenMember := conn.Locals(middlewares.TokenMemberID)
	memberID, ok := tokenMember.(string)
	logger.Log.Info("websocket handle memberID", zap.String("userID", memberID), zap.String("ok", strconv.FormatBool(ok)))

	ticker := time.NewTicker(10 * time.Minute)
	ctxClose, cancel := context.WithCancel(context.Background())

	defer func() {
		ticker.Stop()
		logger.Log.Info("websocket close", zap.String("userID", memberID))
		conn.Close()
		cancel()
	}()

	//client發出close
	//fiber會自動處理(在read msg 回傳err),故需要SetCloseHandler另外接出
	conn.SetCloseHandler(func(code int, text string) error {
		logger.Log.Infof("WebSocket closed:", conn.RemoteAddr())
		return nil
	})

	//server發出ping之後client連線正常會回pong
	//fiber會自動處理回傳pong,故需要SetPongHandler另外接出
	conn.SetPongHandler(func(appData string) error {
		logger.Log.Infof("Received PONG:", appData)
		return nil
	})

	//client發出ping
	//fiber會自動處理ping,故需要SetPingHandler另外接出
	conn.SetPingHandler(func(appData string) error {
		logger.Log.Infof("Received PING:", appData)
		// 如果要手動回 Pong，可以：
		return conn.WriteControl(
			websocket.PongMessage,
			[]byte(appData), // 一般可帶原封不動的資料
			time.Now().Add(time.Second),
		)
	})

	//啟用sub訂閱自己的訊息
	channel := "chat:user:" + memberID
	h.messageUC.memberPubSub.Subscribe(ctxClose, channel, func(resp domain.WSResponse) {
		h.sendResponse(conn, resp)
	})

	// 定期發送 Ping
	go func() {
		for {
			select {
			case <-ticker.C:
				// 發送 Ping 消息
				pingMsg := "ping message"
				if err := conn.WriteMessage(websocket.PingMessage, []byte(pingMsg)); err != nil {
					logger.Log.Errorf("Ping error:", err)
					return
				}
				logger.Log.Infof("%s Ping sent", memberID)
			case <-ctxClose.Done():
				logger.Log.Infof("Ping goroutine cancelled for member:", memberID)
				return
			}
		}
	}()

	for {
		// 1. 讀取前端訊息
		mt, message, err := conn.ReadMessage()
		if err != nil {
			// 檢查是否為 Close 正常結束
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived, //1005 c.WriteMessage(websocket.CloseMessage, []byte{})
			) {
				logger.Log.Errorf("Connection closed:", err)
			} else {
				//直接斷線 1006
				logger.Log.Errorf("websocket read error:", err)
			}
			return
		}
		h.execWebsocketAction(ctx, conn, memberID, mt, message)
	}

}

// 1.	TextMessage
// •	TextMessage 表示文本數據消息。文本消息的負載被解釋為 UTF-8 編碼的文本數據。
// •	使用場景：
// •	用於傳輸純文本數據，例如聊天消息、通知內容或其他以文本形式傳遞的資訊。
// •	在 WebSocket 的應用中，這是最常用的消息類型之一，適用於需要雙向文本通訊的場景。

// 2.	BinaryMessage
// •	BinaryMessage 表示二進制數據消息。
// •	使用場景：
// •	用於傳輸二進制數據，例如圖像、文件、音頻數據或其他非文本的數據。
// •	適用於需要高效傳輸大型數據或非文本數據的應用，例如即時音視頻流、文件共享系統等。

// 3.	CloseMessage
// •	CloseMessage 表示關閉控制消息。可選的消息負載包含一個數字代碼和文本內容。可使用 FormatCloseMessage 函數來格式化關閉消息的負載。
// •	使用場景：
// •	用於通知 WebSocket 連接即將關閉，並提供關閉的原因或狀態碼。
// •	適合在服務器或客戶端完成所有必要操作後，優雅地終止連接的場景。

// 4.	PingMessage
// •	PingMessage 表示 Ping 控制消息。可選的消息負載為 UTF-8 編碼的文本。
// •	使用場景：
// •	用於檢查 WebSocket 連接的健康狀態，確保連接仍然活躍。
// •	服務器或客戶端可定期發送 PingMessage，以檢測另一方是否仍在線。

// 5.	PongMessage
// •	PongMessage 表示 Pong 控制消息。可選的消息負載為 UTF-8 編碼的文本。
// •	使用場景：
// •	用於響應 PingMessage 的消息，表明連接仍然正常。
// •	這是一個 WebSocket 協議中內置的機制，用於維持連接和進行健康檢查。

func (h *ChatWebsocketHandler) execWebsocketAction(ctx context.Context, conn *websocket.Conn, memberID string, mt int, msg []byte) {
	switch mt {
	case websocket.TextMessage:
		h.textMessageAction(ctx, conn, memberID, msg)

	//TODO未使用過BinaryMessage
	// case websocket.BinaryMessage:

	//! close ping pong fiber會自動處理，故需使用setHandler處理
	// case websocket.CloseMessage:
	// 	closeWebSocketConnection(conn, websocket.CloseNormalClosure, "Connection closed by server")

	// case websocket.PingMessage:
	// 	log.Println("Received PingMessage, sending PongMessage")
	// 	if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
	// 		log.Printf("Failed to send PongMessage: %v", err)
	// 	}

	// case websocket.PongMessage:
	// 	log.Println("Received PongMessage, connection is healthy")

	default:
		h.sendError(conn, "unknown action")
	}
}

func (h *ChatWebsocketHandler) textMessageAction(ctx context.Context, conn *websocket.Conn, memberID string, msg []byte) {

	var req domain.WSRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		log.Printf("json unmarshal error: %v", err)
		return
	}

	resp := domain.WSResponse{Action: req.Action, Success: false, Payload: map[string]interface{}{}}
	switch req.Action {
	//單人聊天室邀請
	case string(domain.InvitePrivate):
		invID, err := h.roomUC.ExecuteInvite(ctx, memberID, req.InviteeID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Payload["invitation_id"] = invID
		}
	//單人聊天室邀請同意
	case string(domain.AcceptInvite):
		roomID, err := h.roomUC.ExecuteAccept(ctx, req.InviterID, memberID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Payload["room_id"] = roomID
		}

	//建立群組
	case string(domain.CreateRoom):
		// 直接建立(群組 or private)
		roomID, err := h.roomUC.ExecuteRoom(
			ctx,
			domain.ChatRoomType(req.RoomType),
			req.RoomName,
			[]string{memberID},
			domain.JoinMode(req.JoinMode),
			req.Password,
			req.IsPrivate,
			false,
		)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Payload["room_id"] = roomID
		}

	//加入社群
	case string(domain.JoinRoom):
		// 加入群組
		err := h.roomUC.JoinRoom(ctx, req.RoomID, memberID, req.Password)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
		}

	//離開社群
	case string(domain.ExitRoom):
		err := h.roomUC.ExitRoom(ctx, req.RoomID, memberID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
		}

		// 順便 ephemeralHub.LeaveRoom(req.RoomID, conn) 也可

	//進入聊天室 or 社群
	case string(domain.EnterRoom):
		// ephemeral subscription
		// h.memberHub.EnterRoom(req.RoomID, memberID, conn)

		// 選擇性：查DB 未讀訊息
		msgs, err := h.messageUC.msgRepo.FindEarliestUnread(ctx, memberID, req.RoomID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Payload["unread_messages"] = msgs
		}

		if msgs == nil {
			m, err := h.messageUC.msgRepo.FindMessagesBefore(ctx, req.RoomID, time.Now().Unix())
			if err != nil {
				resp.Success = false
				resp.Error = err.Error()
			}
			resp.Payload["unread_messages"] = ""
			resp.Payload["read_messages"] = m
		}

		ctxEnterRoom, cancel := context.WithCancel(context.Background())
		h.roomCtx = cancel

		// 啟用sub訂閱自己的訊息
		channel := "chat:room:" + req.RoomID
		h.messageUC.memberPubSub.Subscribe(ctxEnterRoom, channel, func(resp domain.WSResponse) {
			h.sendResponse(conn, resp)
		})

	//離開聊天室 or 社群
	case string(domain.LeaveRoom):
		h.roomCtx()
		resp.Success = true
		resp.Payload["leave_room"] = req.RoomID

	//傳送資料
	//message都會寫入db,並傳訊給聊天室內的人
	case string(domain.SendMessage):
		msgID, err := h.messageUC.Execute(ctx, req.RoomID, memberID, req.Content)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Payload["message_id"] = msgID
		}

	//讀取訊息  將未讀訊息改為已讀
	case string(domain.ReadMessage):
		err := h.messageUC.MarkRead(ctx, req.RoomID, req.MessageID, memberID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
		}

	//搜尋所有未讀訊息
	case string(domain.GetUnread):
		msgs, err := h.messageUC.GetCountUnreadMessages(ctx, memberID)
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Success = true
			for _, unread := range msgs {
				resp.Payload[unread.RoomID] = unread.UnreadCount
			}
		}

		//搜尋所有未讀訊息
	case string(domain.GetInvite):
		invitationPending, err := h.roomUC.invRepo.FindInvitationByPending(ctx, memberID)
		if err != nil {
			resp.Error = err.Error()
		}

		if invitationPending != nil {
			for _, inv := range invitationPending {
				resp.Payload[inv.InviterID] = inv.CreatedAt
			}
		}

	default:
		h.sendError(conn, "unknown message types ")
	}

	if resp.Error != "" {
		logger.Log.Error("websocket err ", zap.String("MemberID", memberID), zap.String("Action", req.Action), zap.String("err", resp.Error))
	}
	h.sendResponse(conn, resp)
}

// sendResponse - 發送 JSON 給前端
func (h *ChatWebsocketHandler) sendResponse(conn *websocket.Conn, resp domain.WSResponse) {
	b, _ := json.Marshal(resp)
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		logger.Log.Errorf("write message error:", err)
	}
}

func (h *ChatWebsocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	resp := domain.WSResponse{
		Action:  "error",
		Success: false,
		Payload: map[string]interface{}{
			"error": errorMsg,
		},
	}
	h.sendResponse(conn, resp)
}

func closeWebSocketConnection(conn *websocket.Conn, code int, reason string) {
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason)); err != nil {
		logger.Log.Errorf("Failed to send CloseMessage: %v", err)
	}
	conn.Close()
	logger.Log.Infof("WebSocket connection closed:", conn.RemoteAddr())
}
