package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	// tn, ok := videoThumbnails[videoID]
	// if !ok {
	// 	respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
	// 	return
	// }
	// this was previously gotten from the map - so now we can it from the db
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "video does not exit", err)
	}

	mediaType, base64DataString, err := getMediaTypeFromThumbnailURL(*video.ThumbnailURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal database error", err)
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(base64DataString)))

	_, err = w.Write([]byte(base64DataString))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}

func getMediaTypeFromThumbnailURL(base64String string) (mediaType string,
	base64dataString string, err error) {
	const mediaTypePrefix = "data:"
	const base64Prefix = "base64,"
	dataPortions := strings.Split(base64String, ";")
	if len(dataPortions) != 2 {
		return "", "", fmt.Errorf("error parsing base64String %s", base64String)
	}
	// handle if spolit string does not have 2 parts

	if !strings.HasPrefix(dataPortions[0], mediaTypePrefix) {
		return "", "", fmt.Errorf("error parsing base64String %s", base64String)
	}
	mediaType = strings.TrimPrefix(dataPortions[0], mediaTypePrefix)

	if !strings.HasPrefix(dataPortions[1], base64Prefix) {
		return "", "", fmt.Errorf("error parsing base64String %s", base64String)
	}
	base64dataString = strings.TrimPrefix(dataPortions[1], base64Prefix)
	return mediaType, base64dataString, nil

}
