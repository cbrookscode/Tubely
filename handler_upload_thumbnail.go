package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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
	defer file.Close()

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue getting video from database", err)
		return
	}
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner of the video", err)
		return
	}

	mt, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue parsing media type", err)
		return
	}
	if mt != "image/png" && mt != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "non-image provided", nil)
		return
	}

	uniquePath := filepath.Join(cfg.assetsRoot, videoIDString)
	ext := strings.TrimPrefix(mt, "image/")
	path := fmt.Sprintf(uniquePath + "." + ext)
	fmt.Println(path)
	newFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue creating file", err)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue copying data into new file", err)
		return
	}

	newURL := fmt.Sprintf("http://localhost:8091/%v", path)
	dbVideo.ThumbnailURL = &newURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue updating database with new video info", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
