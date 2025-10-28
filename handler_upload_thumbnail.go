package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"path/filepath"
	"os"
	"io"
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
	// data, err := io.ReadAll(file)
	// if err != nil{
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't read image data", err)
	// 	return
	// }

	videoMeta, err := cfg.db.GetVideo(videoID)
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "couldn't find video", err)
		return
	}

	if videoMeta.UserID !=  userID{
		respondWithError(w, http.StatusUnauthorized,"Not authorized to update ", nil)
		return
	}

	// get file extension
	content := strings.Split(contentType, "/")
	file_ext := content[1]

	// Create a unique filename using the video ID
	videoIDStr := videoID.String()
	log.Printf("VideoIdstr: %s", videoIDStr)

	// Build full file path inside asset directory
	new_path := filepath.Join(cfg.assetsRoot,fmt.Sprintf("%s.%s",videoIDStr,file_ext))
	log.Printf("new file created: %s", new_path)


	// create a new file in the assest directory
	f, err := os.Create(new_path)
	if err != nil{
		respondWithError(w, http.StatusBadRequest, "error occured creating file", err)
		return
	}
	defer f.Close()

	// Copy uploaded file data to the new file
	if _, err := io.Copy(f,file); err != nil{
		respondWithError(w, http.StatusBadRequest, "error occured copying file", err)
		return
	}

	// Construct the public Url for accessing this thumbnail
	url := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port,videoIDStr,file_ext)
	log.Printf("Thumbnail url: %s", url)

	
	// Update the thumbnail URL of the db
	videoMeta.ThumbnailURL = &url


	err = cfg.db.UpdateVideo(videoMeta)
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMeta)
}

// push test