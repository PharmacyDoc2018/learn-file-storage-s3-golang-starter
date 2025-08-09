package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
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
