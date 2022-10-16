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

	"gopkg.in/yaml.v2"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

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
	Language  string
	Items     []ytItem
	RandURL   []string
}

type configYML struct {
	YouTube struct {
		DeveloperKey string
	}
	BasicAuth struct {
		Username string
		Password string
	}
}

var dataBase DB
var config configYML

func main() {
	err := getConfig()
	if err != nil {
		log.Panic(err)
	}
	dataBase, err = initDB()
	if err != nil {
		log.Panic(err)
	}

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
		if len(pair) != 2 || !(pair[0] == config.BasicAuth.Username && pair[1] == config.BasicAuth.Password) {
			authError()
			return
		}
		handler(w, r)
	}
}

func faviconHandler(w http.ResponseWriter, _ *http.Request) {
	file, _ := ioutil.ReadFile("view/favicon.png")
	fmt.Fprint(w, string(file))
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	var p []Rows
	p, _ = dataBase.Select()

	fmap := template.FuncMap{"split": split}

	t := template.Must(template.New("q.html").Funcs(fmap).ParseFiles("./view/q.html"))
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
	languageQ := r.FormValue("language")
	fmap := template.FuncMap{"split": split}

	ytsearch, err := searchItems(q, orderQ, typeQ, channelID, languageQ)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/", 301)
	}
	t := template.Must(template.New("index.html").Funcs(fmap).ParseFiles("./view/index.html"))
	t.Execute(w, ytsearch)
}

func addYTChannel(channelID string) (err error) {
	client := &http.Client{
		Transport: &transport.APIKey{Key: config.YouTube.DeveloperKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return err
	}

	call := service.Channels.List([]string{"snippet"}).Id(channelID)
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

func searchItems(q, orderQ, typeQ, channelIDQ, languageQ string) (ytsearch ytSearch, err error) {
	var call *youtube.SearchListCall
	ytsearch.Query = q
	ytsearch.Order = orderQ
	ytsearch.Type = typeQ
	ytsearch.Language = languageQ
	if channelIDQ != "" {
		ytsearch.ChannelID = channelIDQ
	}

	client := &http.Client{
		Transport: &transport.APIKey{Key: config.YouTube.DeveloperKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return ytsearch, err
	}

	if channelIDQ != "" {
		call = service.Search.List([]string{"snippet"}).
			Q(ytsearch.Query).
			MaxResults(50).
			Order(ytsearch.Order).
			Type(ytsearch.Type).
			ChannelId(ytsearch.ChannelID)
	} else {
		call = service.Search.List([]string{"snippet"}).
			Q(ytsearch.Query).
			MaxResults(50).
			Order(ytsearch.Order).
			Type(ytsearch.Type).
			RelevanceLanguage(ytsearch.Language)
	}

	response, err := call.Do()
	if err != nil {
		return ytsearch, err
	}

	var thumb string

	for _, item := range response.Items {
		if item.Snippet.Thumbnails != nil {
			thumb = item.Snippet.Thumbnails.High.Url
		}

		if ytsearch.Type == "playlist" {
			ytsearch.Items = append(ytsearch.Items, ytItem{
				ID:           item.Id.PlaylistId,
				ChannelTitle: item.Snippet.ChannelTitle,
				Title:        item.Snippet.Title,
				Description:  item.Snippet.Description,
				ThumbURL:     thumb,
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
				ThumbURL:     thumb,
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

func getConfig() (err error) {
	dat, err := ioutil.ReadFile("ytsearch.yml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(dat, &config)
	if err != nil {
		return err
	}
	return nil
}

func split(a, b int) bool {
	return a%b == 0
}
