package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing thumbnail", err)
		return
	}

	tn, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to form thumbnail", err)
	}

	mediaType, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	isJPEG := mediaType == "image/jpeg"
	isPNG := mediaType == "image/png"
	if !isJPEG && !isPNG {
		respondWithError(w, http.StatusBadRequest, "Thumbnail must be jpeg or png filetype", errors.New("wrong tn filetype"))
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

	assetPath := getAssetPath(video.ID, mediaType)
	assetDiskPath := cfg.getAssetDiskPath(assetPath)

	oldPath := strings.Split(*(video.ThumbnailURL), "/")[4]
	oldAssetPath := cfg.getAssetDiskPath(oldPath)

	dst, err := os.Create(assetDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating thumbnail file", err)
		return
	}

	defer dst.Close()
	io.Copy(dst, tn)
	thumbnailURL := cfg.getAssetURL(assetPath)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}

	if _, err := os.Stat(oldAssetPath); err == nil {
		err = os.Remove(oldAssetPath)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error removing old thumbnail", err)
			return
		}

	}
	respondWithJSON(w, http.StatusOK, video)
}
