package errprocess

import (
	"errors"
	"streaming_video_service/pkg/logger"
)

// Set set err info
func Set(errMsg string) error {
	logger.Log.Error(errMsg)
	return errors.New(errMsg)
}
