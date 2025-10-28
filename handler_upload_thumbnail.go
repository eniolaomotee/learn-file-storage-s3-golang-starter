package main

import (
	"fmt"
	"io"
	"net/http"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"encoding/base64"
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

	//Parsing the form data
	const maxMemory = 10 << 20 // 10MB
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil{
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err)
		return 
	}

	defer file.Close()

	// get the media type 
	contentType := header.Header.Get("Content-Type")
	if contentType == ""{
		respondWithError(w, http.StatusBadRequest,"Content-Type header not found", nil)
		return
	}

	// read image data 
	data, err := io.ReadAll(file)
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "Couldn't read image data", err)
		return
	}

	videoMeta, err := cfg.db.GetVideo(videoID)
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "couldn't find video", err)
		return
	}

	if videoMeta.UserID !=  userID{
		respondWithError(w, http.StatusUnauthorized,"Not authorized to update ", nil)
		return
	}

	//convert image data to base64 to be stored in db
	imageData := base64.StdEncoding.EncodeToString(data)

	//create a new data url
	dbMediaUrl := fmt.Sprintf("data:%s;base64,%s", contentType,imageData)

	// storing the url in the thumbnail url of the db
	videoMeta.ThumbnailURL = &dbMediaUrl


	err = cfg.db.UpdateVideo(videoMeta)
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMeta)
}
