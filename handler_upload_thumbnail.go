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

	// 10mb
	const maxMemory = 10 * 1024 * 1024
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue parsing form from", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	medType := header.Header.Get("Content-Type")
	fileInfo, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue reading file from form", err)
		return
	}

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue getting video from database", err)
		return
	}
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner of the video", err)
		return
	}

	videoThumbnails[videoID] = thumbnail{
		data:      fileInfo,
		mediaType: medType,
	}

	newURL := fmt.Sprintf("http://localhost:<port>/api/thumbnails/{%v}", videoID)
	dbVideo.ThumbnailURL = &newURL
	cfg.db.UpdateVideo(dbVideo)
	respondWithJSON(w, http.StatusOK, struct{}{})
}
