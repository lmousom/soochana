package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/lmousom/jisce-soochana/models"
)

var docGlobal *goquery.Document
var allNotice models.Notice
var res []models.Notice

var ai models.AutoInc

func NoticeController() {
	res, err := http.Get("https://jiscollege.ac.in/notice-board.php")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	docGlobal = doc
	if err != nil {
		log.Fatal(err)
	}

}

func (a *API) getAllNotice(ctx context.Context, q string, doc *goquery.Document) ([]models.Notice, bool, error) {
	value, err := a.cache.Get(ctx, q).Result()
	if err == redis.Nil {
		doc.Find(".timeline-left .item").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the title, link and date
			title := s.Find("p").Text()
			link, _ := s.Find("a").Attr("href")
			date := s.Find("h4").Text()
			res = append(res, models.Notice{ai.ID(), strings.TrimSpace(title), "https://jiscollege.ac.in/" + link, date})
		})

		b, err := json.Marshal(res)
		if err != nil {
			return nil, false, err
		}

		// set the value
		err = a.cache.Set(ctx, q, bytes.NewBuffer(b).Bytes(), time.Minute*5).Err()
		if err != nil {
			return nil, false, err
		}

		// return the response
		return res, false, nil
	} else if err != nil {
		fmt.Printf("error calling redis: %v\n", err)
		return nil, false, err
	} else {
		// cache hit
		data := make([]models.Notice, 0)

		// build response
		err := json.Unmarshal(bytes.NewBufferString(value).Bytes(), &data)
		if err != nil {
			return nil, false, err
		}

		// return response
		return data, true, nil
	}

}

func NoticeRouter() *mux.Router {
	api := NewAPI()
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/getNotices", api.GetNoticeTitle).Methods("GET")

	return r
}

func (a *API) GetNoticeTitle(w http.ResponseWriter, r *http.Request) {
	data, cacheHit, err := a.getAllNotice(r.Context(), "notice", docGlobal)

	if err != nil {
		fmt.Printf("error calling JISCE Server: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Cache:   cacheHit,
		Notices: data,
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		fmt.Printf("error encoding response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type API struct {
	cache *redis.Client
}

func NewAPI() *API {
	var opts *redis.Options

	if os.Getenv("LOCAL") == "true" {
		redisAddress := fmt.Sprintf("%s:6379", os.Getenv("REDIS_URL"))
		opts = &redis.Options{
			Addr:     redisAddress,
			Password: "", // no password set
			DB:       0,  // use default DB
		}
	} else {
		builtOpts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
		if err != nil {
			panic(err)
		}
		opts = builtOpts
	}

	rdb := redis.NewClient(opts)

	return &API{
		cache: rdb,
	}
}

type APIResponse struct {
	Cache   bool            `json:"is_cached"`
	Notices []models.Notice `json:"notices"`
}
