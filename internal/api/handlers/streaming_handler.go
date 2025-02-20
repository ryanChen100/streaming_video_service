package handlers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"streaming_video_service/pkg/logger"
	streaming_pb "streaming_video_service/pkg/proto/streaming"
	"time"

	"github.com/gofiber/fiber/v2"
)

// StreamingHandler streaming grpc handler
type StreamingHandler struct {
	StreamingClient streaming_pb.StreamingServiceClient
}

// NewStreamingHandler create streaming handler
func NewStreamingHandler(streamingClient streaming_pb.StreamingServiceClient) *StreamingHandler {
	return &StreamingHandler{
		StreamingClient: streamingClient,
	}
}

// UploadVideo godoc
// @Summary Upload Video via gRPC streaming
// @Description Uploads a video file by first sending video metadata then streaming video chunks
// @Tags Streaming
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "Video Title"
// @Param description formData string true "Video Description"
// @Param type formData string true "Video Type (short or long)"
// @Param file formData file true "Video File"
// @Success 200 {object} streaming_pb.UploadVideoRes "Upload success response"
// @Failure 400 {object} string "Bad Request"
// @Failure 500 {object} string "Internal Server Error"
// @Router /streaming/upload [post]
func (s *StreamingHandler) UploadVideo(c *fiber.Ctx) error {
	// 1. 解析表單資料
	title := c.FormValue("title")
	description := c.FormValue("description")
	videoType := c.FormValue("type")

	// 取得上傳的檔案
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Missing file"})
	}

	// 開啟檔案
	file, err := fileHeader.Open()
	if err != nil {
		logger.Log.Errorf("Open file failed", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer file.Close()

	// 2. 建立 gRPC 流
	grpcCtx := c.UserContext()
	stream, err := s.StreamingClient.UploadVideo(grpcCtx)
	if err != nil {
		logger.Log.Errorf("gRPC UploadVideo stream creation failed", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create gRPC stream"})
	}

	// 3. 發送影片元資料（第一次消息）
	metadata := &streaming_pb.VideoMetadata{
		Title:       title,
		Description: description,
		Type:        videoType,
		FileName:    fileHeader.Filename,
	}
	req := &streaming_pb.UploadVideoReq{
		Data: &streaming_pb.UploadVideoReq_Metadata{
			Metadata: metadata,
		},
	}
	if err := stream.Send(req); err != nil {
		logger.Log.Errorf("Send metadata failed", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to send metadata"})
	}

	// 4. 分段讀取並發送影片檔案資料
	buf := make([]byte, 32*1024) // 32KB chunk
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			logger.Log.Errorf("File read failed", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error reading file"})
		}
		if n == 0 { // 完成
			break
		}
		chunkReq := &streaming_pb.UploadVideoReq{
			Data: &streaming_pb.UploadVideoReq_Chunk{
				Chunk: &streaming_pb.VideoChunk{
					Content: buf[:n],
				},
			},
		}
		if err := stream.Send(chunkReq); err != nil {
			logger.Log.Errorf("Send chunk failed", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error sending file chunk"})
		}
	}

	// 5. 完成後關閉流並接收回應
	res, err := stream.CloseAndRecv()
	if err != nil {
		logger.Log.Errorf("Close and receive failed", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to complete upload"})
	}

	// 6. 返回上傳結果
	return c.JSON(res)
}

// GetVideo godoc
// @Summary Get video streaming info
// @Description Retrieves video streaming info including the HLS URL for playback.
// @Tags Streaming
// @Accept json
// @Produce json
// @Param video_id path string true "Video ID"
// @Success 200 {object} streaming_pb.GetVideoRes "Get video response"
// @Failure 400 {object} string "Bad Request"
// @Failure 404 {object} string "Video not found"
// @Router /streaming/video/{video_id} [get]
func (s *StreamingHandler) GetVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	req := &streaming_pb.GetVideoReq{
		VideoId: videoID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.StreamingClient.GetVideo(ctx, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

// Search godoc
// @Summary Search videos
// @Description Searches for videos by keyword.
// @Tags Streaming
// @Accept json
// @Produce json
// @Param key_word query string true "Search keyword"
// @Success 200 {object} streaming_pb.SearchRes "Search response"
// @Failure 400 {object} string "Bad Request"
// @Router /streaming/search [get]
func (s *StreamingHandler) Search(c *fiber.Ctx) error {
	keyWord := c.Query("key_word")
	req := &streaming_pb.SearchReq{
		KeyWord: keyWord,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.StreamingClient.Search(ctx, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

// GetRecommendations godoc
// @Summary Get recommended videos
// @Description Retrieves recommended videos based on view counts.
// @Tags Streaming
// @Accept json
// @Produce json
// @Param limit query int true "Number of recommendations"
// @Success 200 {object} streaming_pb.GetRecommendationsRes "Recommendations response"
// @Failure 400 {object} string "Bad Request"
// @Router /streaming/recommendations [get]
func (s *StreamingHandler) GetRecommendations(c *fiber.Ctx) error {
	limitStr := c.Query("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid limit"})
	}
	req := &streaming_pb.GetRecommendationsReq{
		Limit: int64(limit),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.StreamingClient.GetRecommendations(ctx, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

// GetIndexM3U8 godoc
// @Summary Get HLS index (m3u8) playlist
// @Description Retrieves the m3u8 playlist file content.
// @Tags Streaming
// @Accept json
// @Produce application/vnd.apple.mpegurl
// @Param video_id path string true "Video ID"
// @Success 200 {string} string "m3u8 playlist content"
// @Failure 400 {object} string "Bad Request"
// @Router /streaming/video/hls/{video_id}/index [get]
func (s *StreamingHandler) GetIndexM3U8(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	req := &streaming_pb.GetIndexM3U8Req{
		VideoId: videoID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.StreamingClient.GetIndexM3U8(ctx, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	return c.Send(res.Content)
}

// GetHlsSegment godoc
// @Summary Get HLS segment (TS file)
// @Description Retrieves a TS segment file content for video streaming.
// @Tags Streaming
// @Accept json
// @Produce video/mp2t
// @Param video_id path string true "Video ID"
// @Param segment path string true "Segment filename"
// @Success 200 {bytes} []byte "TS segment file content"
// @Failure 400 {object} string "Bad Request"
// @Router /streaming/video/hls/{video_id}/{segment} [get]
func (s *StreamingHandler) GetHlsSegment(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	segment := c.Params("segment")
	req := &streaming_pb.GetHlsSegmentReq{
		VideoId: videoID,
		Segment: segment,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.StreamingClient.GetHlsSegment(ctx, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Type", "video/mp2t")
	return c.Send(res.Content)
}
