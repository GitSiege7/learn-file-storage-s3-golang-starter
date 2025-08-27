package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	videoID_str := r.PathValue("videoID")
	if videoID_str == "" {
		respondWithError(w, 400, "No video ID found", nil)
		return
	}

	videoID, err := uuid.Parse(videoID_str)
	if err != nil {
		respondWithError(w, 400, "Invalid video ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No authorization token found", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	meta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 404, "Video not found", err)
		return
	}

	if meta.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid ownership", err)
		return
	}

	mpFile, mpHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 400, "Failed to receive video", err)
		return
	}
	defer mpFile.Close()

	mediaType, _, err := mime.ParseMediaType(mpHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 400, "Failed to parse header", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, 400, "Invalid media type", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 500, "Failed to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, mpFile)
	if err != nil {
		respondWithError(w, 500, "Failed to copy to temp file", err)
		return
	}

	// _, err = tempFile.Seek(0, io.SeekStart)
	// if err != nil {
	// 	respondWithError(w, 500, "Failed to reset file pointer", err)
	// 	return
	// }

	ratio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "Failed to get aspect ratio", err)
		return
	}

	var prefix string
	switch ratio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}

	outputPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "Failed to fast start video", err)
		return
	}

	procVid, err := os.Open(outputPath)
	if err != nil {
		respondWithError(w, 500, "Failed to open processed video", err)
		return
	}
	defer procVid.Close()

	bytes := make([]byte, 32)
	rand.Read(bytes)
	key := fmt.Sprintf("%v/%v.mp4", prefix, base64.RawURLEncoding.EncodeToString(bytes))

	_, err = cfg.s3client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        procVid,
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, 500, "Failed to put object into s3", err)
		return
	}

	url := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, key)
	meta.VideoURL = &url

	cfg.db.UpdateVideo(meta)

	w.WriteHeader(200)
}
