package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(videoID uuid.UUID, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	newIDdata := make([]byte, 32)
	_, err := rand.Read(newIDdata)
	if err != nil {
		log.Printf("error generating asset path")
		return fmt.Sprintf("%s%s", videoID, ext)
	}
	encodedVideoID := base64.URLEncoding.EncodeToString(newIDdata)
	return fmt.Sprintf("%s%s", encodedVideoID, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getVideoAspectRatio(filepath string) (string, error) {
	type videoJsonData struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodexType string `json:"codec_type"`
			Width     int    `json:"width,omitempty"`
			Height    int    `json:"height,omitempty"`
		} `json:"streams"`
	}

	ffprobe := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	var buf []byte
	videoData := bytes.NewBuffer(buf)
	ffprobe.Stdout = videoData
	ffprobe.Run()

	var videoJson videoJsonData
	err := json.Unmarshal(videoData.Bytes(), &videoJson)
	if err != nil {
		return "", err
	}

	var videoWidth int
	var videoHeight int
	for _, stream := range videoJson.Streams {
		if stream.CodexType == "video" {
			videoWidth = stream.Width
			videoHeight = stream.Height
			break
		}
	}

	width := videoWidth
	height := videoHeight

	for i := 9; i > 0; i-- {
		for width%i == 0 && height%i == 0 {
			width /= i
			height /= i
		}
	}
	aspectRatio := fmt.Sprintf("%d:%d", width, height)

	switch aspectRatio {
	case "16:9":
		fallthrough

	case "9:16":
		return aspectRatio, nil

	default:
		return "other", nil
	}

}
