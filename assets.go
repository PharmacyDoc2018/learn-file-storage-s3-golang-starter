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
			CodecType string `json:"codec_type"`
			Width     int    `json:"width,omitempty"`
			Height    int    `json:"height,omitempty"`
		} `json:"streams"`
	}

	ffprobe := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	videoData := &bytes.Buffer{}
	ffprobe.Stdout = videoData
	err := ffprobe.Run()
	if err != nil {
		return "", err
	}
	var videoJson videoJsonData
	err = json.Unmarshal(videoData.Bytes(), &videoJson)
	if err != nil {
		return "", err
	}

	var videoWidth int
	var videoHeight int
	for _, stream := range videoJson.Streams {
		if stream.CodecType == "video" {
			videoWidth = stream.Width
			videoHeight = stream.Height
			break
		}
	}

	ratio := int((videoWidth * 100) / videoHeight)
	ratioCloseToLandscape := ratio >= 175 && ratio <= 177
	rationCloseToPortrait := ratio >= 55 && ratio <= 57

	if ratioCloseToLandscape {
		return "16:9", nil

	} else if rationCloseToPortrait {
		return "9:16", nil

	} else {
		return "other", nil
	}
}

func processVideoForFasterStart(filePath string) (string, error) {
	fmt.Println("FILEPATH:", filePath)
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	newFilepath := base + "-processing" + ext
	fmt.Println("NEW FILE PATH:", newFilepath)
	ffmpeg := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilepath)
	err := ffmpeg.Run()
	if err != nil {
		return "", err
	}
	return newFilepath, nil
}
