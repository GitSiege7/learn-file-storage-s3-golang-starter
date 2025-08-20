package main

import (
	"fmt"
	"io"
	"net/http"

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
	maxMemory := 10 << 20

	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing multipart data", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "Failed form file", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")

	bytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, 500, "Failed to read file data", err)
		return
	}

	meta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Failed to get video", err)
		return
	}

	if meta.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized user", nil)
		return
	}

	videoThumbnails[videoID] = thumbnail{
		data:      bytes,
		mediaType: mediaType,
	}

	thumbURL := fmt.Sprintf("http://localhost:8091/api/thumbnails/%v", videoID)

	meta.ThumbnailURL = &thumbURL

	cfg.db.UpdateVideo(meta)

	respondWithJSON(w, http.StatusOK, meta)
}
