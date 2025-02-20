package app

import (
	"bytes"
	"context"
	"io"

	"streaming_video_service/internal/streaming/domain"
	streaming_pb "streaming_video_service/pkg/proto/streaming"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamingGRPCServer 用來實作 StreamingGRPCServer
type StreamingGRPCServer struct {
	streaming_pb.UnimplementedStreamingServiceServer
	Usecase StreamingUseCase
}

// UploadVideo 實作 上傳影片
// UploadVideo 實作客戶端流式 RPC 方法
func (s *StreamingGRPCServer) UploadVideo(stream streaming_pb.StreamingService_UploadVideoServer) error {
	var metadata *streaming_pb.VideoMetadata
	var fileBuffer bytes.Buffer

	// 循環接收來自客戶端的流
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 流結束，退出循環
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "接收流失敗: %v", err)
		}

		// 根據 oneof 欄位判斷數據類型
		switch data := req.Data.(type) {
		case *streaming_pb.UploadVideoReq_Metadata:
			// 第一次應傳送元資料
			metadata = data.Metadata
		case *streaming_pb.UploadVideoReq_Chunk:
			// 後續傳送檔案區塊
			_, err := fileBuffer.Write(data.Chunk.Content)
			if err != nil {
				return status.Errorf(codes.Internal, "寫入檔案區塊失敗: %v", err)
			}
		default:
			return status.Errorf(codes.InvalidArgument, "未知的數據類型")
		}
	}

	// 檢查必須的元資料是否存在
	if metadata == nil {
		return status.Errorf(codes.InvalidArgument, "缺少影片元資料")
	}

	// 調用 usecase 層進行上傳處理
	upRes, err := s.Usecase.UploadVideo(domain.UploadVideoReq{
		Title:       metadata.Title,
		Description: metadata.Description,
		Type:        metadata.Type,
		FileName:    metadata.FileName, // 客戶端應提供檔案名稱
		File:        bytes.NewReader(fileBuffer.Bytes()),
	})
	if err != nil {
		// 返回錯誤回應
		res := &streaming_pb.UploadVideoRes{
			Success: false,
			Message: err.Error(),
		}
		return stream.SendAndClose(res)
	}

	// 返回成功回應
	res := &streaming_pb.UploadVideoRes{
		Success: true,
		Message: upRes.Message,
		VideoId: int64(upRes.VideoID),
	}
	return stream.SendAndClose(res)
}

// GetVideo 實作 依video id取得 video
func (s *StreamingGRPCServer) GetVideo(ctx context.Context, req *streaming_pb.GetVideoReq) (*streaming_pb.GetVideoRes, error) {
	video, err := s.Usecase.GetVideo(req.VideoId)
	if err != nil {
		return &streaming_pb.GetVideoRes{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	return &streaming_pb.GetVideoRes{
		Success: true,
		VideoId: int64(video.VideoID),
		Title:   video.Title,
		HlsUrl:  video.HlsURL,
	}, nil
}

// Search 實作 Search
func (s *StreamingGRPCServer) Search(ctx context.Context, req *streaming_pb.SearchReq) (*streaming_pb.SearchRes, error) {
	videos, err := s.Usecase.Search(req.KeyWord)
	if err != nil {
		return &streaming_pb.SearchRes{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	videoRes := make([]*streaming_pb.SearchFeedBack, len(videos))
	for _, video := range videos {
		videoRes = append(videoRes, &streaming_pb.SearchFeedBack{
			VideoId:     int64(video.ID),
			Title:       video.Title,
			Description: video.Description,
			FileName:    video.FileName, // 存於 MinIO 上的 object key
			Type:        video.Type,
			Status:      video.Status, // "uploaded", "processing", "ready"
			ViewCCount:  int64(video.ViewCount),
		})
	}
	return &streaming_pb.SearchRes{
		Success: true,
		Video:   videoRes,
	}, nil
}

// GetRecommendations 實作 取得最熱門video
func (s *StreamingGRPCServer) GetRecommendations(ctx context.Context, req *streaming_pb.GetRecommendationsReq) (*streaming_pb.GetRecommendationsRes, error) {
	videos, err := s.Usecase.GetRecommendations(int(req.Limit))
	if err != nil {
		return &streaming_pb.GetRecommendationsRes{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	videoRes := make([]*streaming_pb.SearchFeedBack, len(videos))
	for _, video := range videos {
		videoRes = append(videoRes, &streaming_pb.SearchFeedBack{
			VideoId:     int64(video.ID),
			Title:       video.Title,
			Description: video.Description,
			FileName:    video.FileName, // 存於 MinIO 上的 object key
			Type:        video.Type,
			Status:      video.Status, // "uploaded", "processing", "ready"
			ViewCCount:  int64(video.ViewCount),
		})
	}
	return &streaming_pb.GetRecommendationsRes{
		Success: true,
		Video:   videoRes,
	}, nil
}

// GetIndexM3U8 實作 取得m3u8
func (s *StreamingGRPCServer) GetIndexM3U8(ctx context.Context, req *streaming_pb.GetIndexM3U8Req) (*streaming_pb.GetIndexM3U8Res, error) {
	m3u8, err := s.Usecase.GetIndexM3U8(ctx, req.VideoId)
	if err != nil {
		return &streaming_pb.GetIndexM3U8Res{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &streaming_pb.GetIndexM3U8Res{
		Success: true,
		Content: m3u8,
	}, nil
}

// GetHlsSegment 實作 依video id & segment 讀取 ts
func (s *StreamingGRPCServer) GetHlsSegment(ctx context.Context, req *streaming_pb.GetHlsSegmentReq) (*streaming_pb.GetHlsSegmentRes, error) {
	ts, err := s.Usecase.GetHlsSegment(ctx, req.VideoId, req.Segment)
	if err != nil {
		return &streaming_pb.GetHlsSegmentRes{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	return &streaming_pb.GetHlsSegmentRes{
		Success: true,
		Content: ts,
	}, nil
}
