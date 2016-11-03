package main

import (
	"html/template"
	"log"
	"net/http"

	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

const developerKey = "AIzaSyDnNqubCXqrNHD8w_GHsyRK7X6GU-k4MzU"

type YTItem struct {
	ID           string
	ChannelTitle string
	Title        string
	Description  string
	ThumbURL     string
	ChannelID    string
}

type YTSearch struct {
	Query string
	Items []YTItem
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/q", viewHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var p YTSearch
	t, _ := template.ParseFiles("./view/q.html")
	t.Execute(w, p)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("query")
	ytsearch := ytSearch(q)
	t, _ := template.ParseFiles("./view/index.html")
	t.Execute(w, ytsearch)
}

func ytSearch(q string) (ytsearch YTSearch) {
	ytsearch.Query = q

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatal(err)
	}

	call := service.Search.List("snippet").
		Q(ytsearch.Query).
		MaxResults(50).
		Order("relevance").
		Type("playlist")

	response, err := call.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range response.Items {

		ytsearch.Items = append(ytsearch.Items, YTItem{
			ID:           item.Id.PlaylistId,
			ChannelTitle: item.Snippet.ChannelTitle,
			Title:        item.Snippet.Title,
			Description:  item.Snippet.Description,
			ThumbURL:     item.Snippet.Thumbnails.High.Url,
			ChannelID:    item.Snippet.ChannelId,
		})
	}

	return ytsearch
}
