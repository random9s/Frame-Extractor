package main

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Metadata struct {
	Title string
	URL   string
	Path  string
	Error string
}

const MAX_VIDEO_SIZE int64 = 10000000
const FILE_TYPE_ERR string = "1"
const FILE_SIZE_ERR string = "2"

func Index(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.RawQuery, "=") {
		path := strings.Split(r.URL.RawQuery, "")
		error_num := path[6:7]

		var meta *Metadata

		if error_num[0] == FILE_TYPE_ERR {
			meta = &Metadata{Error: "Incorrect file type"}
		} else if error_num[0] == FILE_SIZE_ERR {
			meta = &Metadata{Error: "File size too large"}
		} else {
			http.Redirect(w, r, "/", 301)
		}

		t, _ := template.ParseFiles("views/index.html")
		t.Execute(w, meta)

		return
	}

	var meta = &Metadata{Title: "Index"}
	t, _ := template.ParseFiles("views/index.html")
	t.Execute(w, meta)
}

func ConvertVideoToImage(w http.ResponseWriter, r *http.Request) {
	//Create file
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	contentType := strings.Join(handler.Header["Content-Type"], "")
	valid := CheckIfValidFileType(contentType)

	contentLength := r.ContentLength

	if valid && contentLength <= MAX_VIDEO_SIZE {

		//Create unique directory
		tempDirName, err := ioutil.TempDir("temps", "video")
		if err != nil {
			os.Remove(tempDirName)
			http.Error(w, err.Error(), 500)
			return
		}

		tempDirSlice := strings.Split(tempDirName, "/")
		setId := tempDirSlice[1]

		//Save video to unique directory
		f, err := os.OpenFile("./temps/"+setId+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), 500)
			return
		}

		defer f.Close()
		io.Copy(f, file)

		//Create safe url
		url := "image_sets/" + setId
		var meta = &Metadata{URL: url}

		c := make(chan string)
		go VideoToImage(setId, handler.Filename, c)

		t, _ := template.ParseFiles("views/imageSets.html")
		t.Execute(w, meta)

		return
	}

	if !valid {
		http.Redirect(w, r, "/?error=1", 301)
	}

	if contentLength > MAX_VIDEO_SIZE {
		http.Redirect(w, r, "/?error=2", 301)
	}
}

func GetImageSet(w http.ResponseWriter, r *http.Request) {
	setId := r.URL.Path[12:]
	var path = html.EscapeString(setId)
	url := "temps/" + path + ".zip"

	videoId := strings.Split(r.URL.Path, "/")
	var video_path = html.EscapeString(videoId[2])

	var meta = &Metadata{
		URL:  url,
		Path: video_path,
	}

	t, _ := template.ParseFiles("views/download.html")
	t.Execute(w, meta)
}

func CheckIfDone(w http.ResponseWriter, r *http.Request) {
	setId := strings.Split(r.URL.Path, "/")
	var path = html.EscapeString(setId[2])

	if _, err := os.Stat("./temps/" + path + "/done.txt"); err == nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode("true"); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode("false"); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	return
}

func CheckIfValidFileType(contentType string) bool {
	validTypes := []string{"video/x-flv",
		"video/mp4",
		"application/x-mpegURL",
		"video/MP2T",
		"video/3gpp",
		"video/quicktime",
		"video/x-msvideo",
		"video/x-ms-wmv"}

	for _, c_type := range validTypes {
		if contentType == c_type {
			return true
		}
	}
	return false
}
