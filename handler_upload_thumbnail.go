package main

import (
	"fmt"
	"io"
	"net/http"
	"log"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	log.Println("hello guys")
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

	log.Printf("file is %s",contentType)


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

	// save thumbnail in global map
	videoThumbnails[videoID] = thumbnail{
		data: data,
		mediaType: contentType,
	}

	// update video metadata
	url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s",cfg.port, videoID)
	videoMeta.ThumbnailURL = &url
	

	err = cfg.db.UpdateVideo(videoMeta)
	if err != nil{
		delete(videoThumbnails,videoID)
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMeta)
}
