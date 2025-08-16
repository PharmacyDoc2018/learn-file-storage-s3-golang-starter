package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory) // 1GB
	defer r.Body.Close()

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing video", err)
		return
	}

	videoMultiPart, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to form video", err)
	}
	defer videoMultiPart.Close()

	mediaType, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Video must be mp4 filetype", errors.New("wrong tn filetype"))
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", errors.New("401 unauthorized"))
		return
	}

	dst, err := os.CreateTemp("", "tubely-upload*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating temp file", err)
		return
	}
	defer os.Remove(dst.Name())
	defer dst.Close()
	_, err = io.Copy(dst, videoMultiPart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving video to disk", err)
		return
	}

	dst.Seek(0, io.SeekStart)

	dstPath := dst.Name()
	aspectRatio, err := getVideoAspectRatio(dstPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error determining aspect ratio", err)
		return
	}

	var keyPrefix string
	switch aspectRatio {
	case "16:9":
		keyPrefix = "landscape"

	case "9:16":
		keyPrefix = "portrait"

	default:
		keyPrefix = "other"
	}

	newIDdata := make([]byte, 32)
	_, err = rand.Read(newIDdata)
	if err != nil {
		log.Printf("error generating asset path")
		respondWithError(w, http.StatusInternalServerError, "Error generating bucket key", err)
		return
	}

	bucketKey := keyPrefix + "/" + base64.URLEncoding.EncodeToString(newIDdata)
	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &bucketKey,
		Body:        dst,
		ContentType: &mediaType,
	}
	log.Printf("Uploading video to S3: %s", bucketKey)
	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		log.Printf("Upload error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Error uploading video", err)
		return
	}

	videoURL := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + bucketKey
	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}
}
