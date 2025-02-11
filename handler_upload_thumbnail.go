package main

import (
	"errors"
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
	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "problem with parsing of form", err)
		return
	}
	/*
		header that is returned from FormFile is of type multiPart.FileHeader
		FileHeader itself as Header filed which is of type textproto.MIMEheader which is essentially a map of key value pairs

		### the MIMEtype of each file will be stored in the file header (which is in fileHeader in the above example)
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

	mediaType, err := parseMediaType(fileHeader.Header.Get("Content-Type"))

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file type not valid", err)
		return
	}
	fileExtension, err := getExtensionFromImgMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file type not valid", err)
		return
	}
	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "error reading file", err)
	// 	return
	// }
	fileName := fmt.Sprintf("%s.%s", videoIDString, fileExtension)
	fullFilePath := filepath.Join(cfg.assetsRoot, fileName)
	fmt.Printf("cfg.assetsRoot: %s\n", cfg.assetsRoot)
	fmt.Printf("fullFilePath: %s\n", fullFilePath)
	fileToDisk, err := os.Create(fullFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing file", err)
		return
	}
	defer fileToDisk.Close()

	io.Copy(fileToDisk, file)

	updatedThumbnailURL := generateFileUrl(cfg.port, videoIDString, fileExtension)
	fmt.Printf("updatedThumbnailURL: %s\n", updatedThumbnailURL)
	video.ThumbnailURL = &updatedThumbnailURL
	dbErr := cfg.db.UpdateVideo(video)
	if dbErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error updated video in database", dbErr)
		return
	}

	// videoThumbnails[videoID] = thumbnail{
	// 	data:      data,
	// 	mediaType: mediaType,
	// }

	respondWithJSON(w, http.StatusOK, video)
}

//	func generateDataURL(mediaType string, dataBase64 string) string {
//		return fmt.Sprintf("data:%s;base64,%s", mediaType, dataBase64)
//	}
func generateFileUrl(port string, videoID string, fileExtension string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s.%s",
		port,
		videoID,
		fileExtension,
	)
}

func getExtensionFromImgMediaType(mediaTypeString string) (string, error) {
	if !strings.HasPrefix(mediaTypeString, "image/") {
		return "", errors.New("media type is not an image")
	}
	return strings.TrimPrefix(mediaTypeString, "image/"), nil
}

func parseMediaType(header string) (mediatype string, err error) {
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return "", err
	}
	return mediaType, nil
}
