package main

import (
	"errors"
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
	//this part of function gets the value from the url passed in, and then attempts to parse to UUID. if fails, returns with error

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	//if token is not found, then GetBearerToken will always throw error

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	//validity of token will be implicit - this would include any time durations that are specified.

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	/*
		bit shift will shift the value of 10 in decimal (1010) by 20 spaces to the right
		each move to the right will multiply the current number by 2 (due to how binary numbers work)
		this essentially is making maxMemory worth 10 megabytes of memory
	*/
	parseErr := r.ParseMultipartForm(maxMemory)
	if parseErr != nil {
		respondWithError(w, http.StatusBadRequest, "problem with parsing of form", parseErr)
		return
	}
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "problem with parsing of form", err)
		return
	}
	/*
		header that is returned from FormFile is of type multiPart.FileHeader
		FileHeader itself as Header filed which is of type textproto.MIMEheader which is essentially a map of key value pairs
	*/
	video, err := cfg.db.GetVideo(videoID)
	// getting the data in the database relating to the error
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "video record could not be found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user is not authorized", errors.New("user is not authorized"))
		return
	}
	//checks to see if the user logged in has accesst to the video, if not returns unauthorized error

	mediaType := header.Header.Get("Content-Type")

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading file", err)
		return
	}
	updatedThumbnailURL := generateThumbnailURL(cfg, videoIDString)
	video.ThumbnailURL = &updatedThumbnailURL
	dbErr := cfg.db.UpdateVideo(video)
	if dbErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error updated video in database", dbErr)
		return
	}

	videoThumbnails[videoID] = thumbnail{
		data:      data,
		mediaType: mediaType,
	}
	respondWithJSON(w, http.StatusOK, video)
}

func generateThumbnailURL(cfg *apiConfig, videoId string) string {
	return fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoId)
}
