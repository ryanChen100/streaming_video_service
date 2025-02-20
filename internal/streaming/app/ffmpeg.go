package app

import (
	"fmt"
	"log"
	"os/exec"
)

// TranscodeToHLS 將 inputPath 轉成 HLS 格式，輸出到 outputDir（會產生 index.m3u8 與 TS 分段）
func TranscodeToHLS(inputPath, outputDir string) error {
	cmdArgs := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "0",
		fmt.Sprintf("%s/index.m3u8", outputDir),
	}
	log.Printf("執行 FFmpeg HLS: ffmpeg %v", cmdArgs)
	cmd := exec.Command("ffmpeg", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg HLS 錯誤: %v, output: %s", err, string(output))
	}
	return nil
}

// TranscodeToDASH 將 inputPath 轉成 DASH 格式，輸出到 outputDir（會產生 manifest.mpd）
func TranscodeToDASH(inputPath, outputDir string) error {
	outputMPD := fmt.Sprintf("%s/manifest.mpd", outputDir)
	cmdArgs := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-f", "dash",
		outputMPD,
	}
	log.Printf("執行 FFmpeg DASH: ffmpeg %v", cmdArgs)
	cmd := exec.Command("ffmpeg", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg DASH 錯誤: %v, output: %s", err, string(output))
	}
	return nil
}
