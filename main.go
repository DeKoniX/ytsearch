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

type ytItem struct {
	ID           string
	ChannelTitle string
	Title        string
	Description  string
	ThumbURL     string
	ChannelID    string
	PublishedAT  string
	URL          string
}

type ytSearch struct {
	Query     string
	Order     string
	Type      string
	ChannelID string
	Items     []ytItem
	RandURL   []string
}

var dataBase DB

func main() {
	dataBase, _ = initDB()

	fs := http.FileServer(http.Dir("./view/static"))
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.HandleFunc("/", basicAuth(indexHandler))
	http.HandleFunc("/q", basicAuth(viewHandler))
	http.HandleFunc("/channeladd", basicAuth(channelADDHandler))
	http.HandleFunc("/channeldelete", basicAuth(channelDeleteHandler))
	http.HandleFunc("/favicon.png", faviconHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := "dekonix"
		password := "hamster"
		authError := func() {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"YTSearch\"")
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

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile("view/favicon.png")
	fmt.Fprint(w, string(file))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var p []Rows
	p, _ = dataBase.Select()
	t, _ := template.ParseFiles("./view/q.html")
	t.Execute(w, p)
}

func channelADDHandler(w http.ResponseWriter, r *http.Request) {
	channelID := r.FormValue("channel_id")
	err := addYTChannel(channelID)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", 301)
}

func channelDeleteHandler(w http.ResponseWriter, r *http.Request) {
	channelID := r.FormValue("channelid")
	err := dataBase.Delete(channelID)
	if err != nil {
		log.Println("DataBase Delete: ", err)
	}
	http.Redirect(w, r, "/", 301)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("query")
	orderQ := r.FormValue("order")
	typeQ := r.FormValue("type")
	channelID := r.FormValue("channelID")

	ytsearch, err := searchItems(q, orderQ, typeQ, channelID)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/", 301)
	}
	t, _ := template.ParseFiles("./view/index.html")
	t.Execute(w, ytsearch)
}

func addYTChannel(channelID string) (err error) {
	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return err
	}

	call := service.Channels.List("snippet").Id(channelID)
	response, err := call.Do()
	if err != nil {
		return err
	}

	item := response.Items[0].Snippet

	_, err = dataBase.Insert(channelID, item.Title, item.Description, item.Thumbnails.High.Url)
	if err != nil {
		return err
	}
	return nil
}

func searchItems(q, orderQ, typeQ, channelIDQ string) (ytsearch ytSearch, err error) {
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
		return ytsearch, err
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
		return ytsearch, err
	}

	for _, item := range response.Items {
		if ytsearch.Type == "playlist" {
			ytsearch.Items = append(ytsearch.Items, ytItem{
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
			ytsearch.Items = append(ytsearch.Items, ytItem{
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

	ytsearch.RandURL = thisIsRandURL(ytsearch.Items)

	return ytsearch, nil
}

func thisIsRandURL(items []ytItem) (itemsURL []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randPerm := r.Perm(len(items))
	for _, randID := range randPerm {
		itemsURL = append(itemsURL, items[randID].URL)
	}

	return itemsURL
}
