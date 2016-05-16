package main

import (
	"encoding/csv"
	"github.com/ant0ine/go-json-rest/rest"
	"io"
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type User struct {
	Id   string
	Name string
	Surname string
}

type Hotkey struct {
	Order       int    `json:"order,omitempty"`
	Name        string `json:"name,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Group       string `json:"group,omitempty"`
	Type        string `json:"type,omitempty"`
	Url         string `json:"url,omitempty"`
	Shortcut    string `json:"shortcut,int64,omitempty"`
	Description string `json:"description,omitempty"`
}

func GetHotkeys(w rest.ResponseWriter, r *rest.Request) {
	ctx := appengine.NewContext(r.Request)

	q := datastore.NewQuery("Hotkey")

	name := r.URL.Query().Get("name")
	if name != "" {
		q = q.Filter("Name =", name)
	}
	platform := r.URL.Query().Get("platform")
	if platform != "" {
		q = q.Filter("Platform =", platform)
	}
	group := r.URL.Query().Get("group")
	if group != "" {
		q = q.Filter("Group =", group)
	}
	_type := r.URL.Query().Get("type")
	if _type != "" {
		q = q.Filter("Type =", _type)
	}
	url := r.URL.Query().Get("url")
	if url != "" {
		q = q.Filter("Url =", url)
	}
	var hotkeys []Hotkey
	q = q.Order("Order")
	_, err := q.GetAll(ctx, &hotkeys)
	if err != nil {
		log.Errorf(ctx, "", err)
	}

	if hotkeys != nil {
		w.WriteJson(&hotkeys)
	} else {
		w.WriteJson([]int{})
	}
}

func GetHotkey(w rest.ResponseWriter, r *rest.Request) {
	user := User{
		Id:   r.PathParam("id"),
		Name: "code",
		Surname: r.URL.Query().Get("code"),
	}
	w.WriteJson(&user)
}

func PostHotkey(w rest.ResponseWriter, r *rest.Request) {
	ctx := appengine.NewContext(r.Request)

	hotkey := Hotkey{}
	err := r.DecodeJsonPayload(&hotkey)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "Hotkey", nil), &hotkey)

	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteJson(&hotkey)
	/* w.WriteJson(map[string]string{"Body": "Hello World!"}) */
}

func BulkInsertHotkeys(w rest.ResponseWriter, r *rest.Request) {
	ctx := appengine.NewContext(r.Request)

	hotkeys := []Hotkey{}
	err := r.DecodeJsonPayload(&hotkeys)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := datastore.NewQuery("Hotkey")
	var deleteHotkeys []Hotkey
	deleteKeys, err := q.GetAll(ctx, &deleteHotkeys)
	if err != nil {
		log.Errorf(ctx, "", err)
	}
	err = datastore.DeleteMulti(ctx, deleteKeys)

	var keys []*datastore.Key
	for range hotkeys {
		keys = append(keys, datastore.NewIncompleteKey(ctx, "Hotkey", nil))
	}

	_, err = datastore.PutMulti(ctx, keys, hotkeys)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteJson(&hotkeys)
}

func PullUpdateHotkeys(w rest.ResponseWriter, r *rest.Request) {
	ctx := appengine.NewContext(r.Request)

	downloadUrl := "http://docs.google.com/spreadsheets/d/1JH-eQdWAXx70T5XkGTfz4jgXTMvG9Fpm96ANnyRpnkQ/pub?gid=0&single=true&output=csv"
	// downloadUrl := "https://spreadsheets.google.com/feeds/list/1JH-eQdWAXx70T5XkGTfz4jgXTMvG9Fpm96ANnyRpnkQ/od6/public/values?alt=json"
	log.Infof(ctx, "DownloadUrl %s", downloadUrl)
	body := downloadFromUrl(ctx, downloadUrl)

	csvReader := csv.NewReader(strings.NewReader(body))
	csvReader.Comma = ','
	hotkeys := []Hotkey{}

	var hotkey Hotkey

	for {
		err := Unmarshal(csvReader, &hotkey)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf(ctx, "Error while parsing csv", err)
		}

		if hotkey.Name == "" {
			continue
		}

		log.Infof(ctx, "Saw record %T", hotkey)
		log.Infof(ctx, "Saw record %v", hotkey)
		hotkeys = append(hotkeys, hotkey)
	}

	q := datastore.NewQuery("Hotkey")
	var deleteHotkeys []Hotkey
	deleteKeys, err := q.GetAll(ctx, &deleteHotkeys)
	if err != nil {
		log.Errorf(ctx, "", err)
	}
	err = datastore.DeleteMulti(ctx, deleteKeys)

	var keys []*datastore.Key
	for range hotkeys {
		keys = append(keys, datastore.NewIncompleteKey(ctx, "Hotkey", nil))
	}

	for i := 0; i < len(hotkeys) / 500; i++ {
		_, err = datastore.PutMulti(ctx, keys, hotkeys[i: (i + 1) * 500])
		if err != nil {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteJson(&hotkeys)
}

func init() {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,
		OriginValidator: func(origin string, request *rest.Request) bool {
			return true
			/* return origin == "http://localhost:3000" */
		},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{
			"Accept", "Content-Type", "X-Custom-Header", "Origin"},
			AccessControlAllowCredentials: true,
			AccessControlMaxAge:           3600,
		})
		router, err := rest.MakeRouter(
			rest.Get("/api/hotkeys/:id", GetHotkey),
			rest.Get("/api/hotkeys/", GetHotkeys),
			rest.Post("/api/hotkeys/", PostHotkey),
			rest.Post("/api/hotkeys/bulk_insert/", BulkInsertHotkeys),
			rest.Get("/api/hotkeys/pull_update/", PullUpdateHotkeys),
		)
		if err != nil {
		}
		api.SetApp(router)
		http.Handle("/", api.MakeHandler())
	}
