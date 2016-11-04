package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

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
	PublishedAT  string
	URL          string
}

type YTSearch struct {
	Query     string
	Order     string
	Type      string
	ChannelID string
	Items     []YTItem
	RandURL   []string
}

func main() {
	http.HandleFunc("/", BasicAuth(indexHandler))
	http.HandleFunc("/q", BasicAuth(viewHandler))
	http.HandleFunc("/index.js", jsHandler)
	http.HandleFunc("/favicon.png", faviconHandler)
	http.ListenAndServe(":8080", nil)
}

func BasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := "dekonix"
		password := "hamster"
		authError := func() {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Zork\"")
			http.Error(w, "authorization failed", http.StatusUnauthorized)
		}
		auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			authError()
			return
		}
		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			authError()
			return
		}
		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 || !(pair[0] == username && pair[1] == password) {
			authError()
			return
		}
		handler(w, r)
	}
}

func jsHandler(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile("view/index.js")
	fmt.Fprint(w, string(file))
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile("view/favicon.png")
	fmt.Fprint(w, string(file))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var p YTSearch
	t, _ := template.ParseFiles("./view/q.html")
	t.Execute(w, p)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("query")
	orderQ := r.FormValue("order")
	typeQ := r.FormValue("type")
	channelID := r.FormValue("channelID")

	ytsearch := ytSearch(q, orderQ, typeQ, channelID)
	t, _ := template.ParseFiles("./view/index.html")
	t.Execute(w, ytsearch)
}

func ytSearch(q, orderQ, typeQ, channelIDQ string) (ytsearch YTSearch) {
	var call *youtube.SearchListCall
	ytsearch.Query = q
	ytsearch.Order = orderQ
	ytsearch.Type = typeQ
	if channelIDQ != "" {
		ytsearch.ChannelID = channelIDQ
	}

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatal(err)
	}

	if channelIDQ != "" {
		call = service.Search.List("snippet").
			Q(ytsearch.Query).
			MaxResults(50).
			Order(ytsearch.Order).
			Type(ytsearch.Type).
			ChannelId(ytsearch.ChannelID)
	} else {
		call = service.Search.List("snippet").
			Q(ytsearch.Query).
			MaxResults(50).
			Order(ytsearch.Order).
			Type(ytsearch.Type)
	}

	response, err := call.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range response.Items {
		if ytsearch.Type == "playlist" {
			ytsearch.Items = append(ytsearch.Items, YTItem{
				ID:           item.Id.PlaylistId,
				ChannelTitle: item.Snippet.ChannelTitle,
				Title:        item.Snippet.Title,
				Description:  item.Snippet.Description,
				ThumbURL:     item.Snippet.Thumbnails.High.Url,
				ChannelID:    item.Snippet.ChannelId,
				PublishedAT:  item.Snippet.PublishedAt,
				URL:          "https://www.youtube.com/playlist?list=" + item.Id.PlaylistId,
			})
		} else {
			ytsearch.Items = append(ytsearch.Items, YTItem{
				ID:           item.Id.VideoId,
				ChannelTitle: item.Snippet.ChannelTitle,
				Title:        item.Snippet.Title,
				Description:  item.Snippet.Description,
				ThumbURL:     item.Snippet.Thumbnails.High.Url,
				ChannelID:    item.Snippet.ChannelId,
				PublishedAT:  item.Snippet.PublishedAt,
				URL:          "https://www.youtube.com/watch?v=" + item.Id.VideoId,
			})
		}
	}

	ytsearch.RandURL = ThisIsRandUrl(ytsearch.Items)

	return ytsearch
}

func ThisIsRandUrl(items []YTItem) (itemsURL []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randPerm := r.Perm(len(items))
	for _, randID := range randPerm {
		itemsURL = append(itemsURL, items[randID].URL)
	}

	return itemsURL
}
