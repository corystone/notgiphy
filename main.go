package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

/* This handler sets all the required headers for CORS to work from localhost, for  UI development. */
func openCORS(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
		if r.Method == "OPTIONS" {
			fmt.Fprintln(w, "")
			return
		}
		next.ServeHTTP(w, r)
	}
}

/* Print the request to stdout. */
func requestLog(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("REQUEST, Method: %v, URL: %v\n", r.Method, r.URL)
		fmt.Printf("FULL REQUEST: %+v\n", r)
		next.ServeHTTP(w, r)
	}
}

/* Cheecks to see if the cookie is valid. Sets the user in the request context for downstream handelers. */
func authRequired(next http.Handler, db Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/api/auth") || r.URL.Path == "/api/gifs" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("sessionid")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Printf("Cookie error: %+v\n", err)
			return
		}
		sessionid := cookie.Value
		user, dberr := db.SessionGet(sessionid)
		if dberr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Printf("SessionGet error: %+v\n", dberr)
			return
		}
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

/* If the path exists in the static directory, send that file.
 * This lets us boostrap the angular app's static files. */
func sendFile(s string, w http.ResponseWriter) bool {
	prefix := "./static"
	fullpath := path.Clean(prefix + s)
	stat, err := os.Stat(fullpath)
	if err == nil && stat.Mode().IsRegular() {
		f, err := os.Open(fullpath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("ERROR os.Open: %+v\n", err)
			fmt.Fprintln(w, "Internal Server Error")
			return true
		}
		/* Browsers seem to hate it if you send css text/plain. */
		if strings.HasSuffix(fullpath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}
		if _, err := io.Copy(w, f); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("ERROR io.Copy: %+v\n", err)
			fmt.Fprintln(w, "Internal Server Error")
			return true
		}
		return true
	}
	return false
}

/* If not an api call, try sending the path from static. */
func staticHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api") && sendFile(r.URL.Path, w) {
			return
		}
		next.ServeHTTP(w, r)
	}
}

/* Last handler handles gif queries and falls back to sending index.html. */
func rootHandler(db Db, client *GifClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.ParseForm()
		q := r.FormValue("q")
		if q == "" {
			if !sendFile("/index.html", w) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Internal Server Error")
				fmt.Printf("Couldnt send static index.html\n")
			}
			return
		}
		page := getPage(r.FormValue("p"))
		gifs, err := client.Search(q, page)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal Server Error")
			fmt.Printf("Error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gifs)
	}
}

/* Gets a single gif. */
func gifsHandler(client *GifClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Missing id")
			return
		}
		gif, err := client.Get(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("client error: %+v\n", err)
			return
		}
		if gif == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gif)
	}
}

func getUser(r *http.Request) string {
	if user := r.Context().Value("user"); user != nil {
		return user.(string)
	}
	fmt.Printf("ERROR: No USER for authed call! (Shouldn't happen...)\n")
	return ""
}

func getPage(p string) int {
	page, err := strconv.Atoi(p)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

/* Adding/removing favorites. Getting favorites lists, full and by tag. */
func favoritesHandler(db Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUser(r)
		if user == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.ParseForm()
		if r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			var gif Gif
			err := decoder.Decode(&gif)
			if err != nil {
				fmt.Printf("Json decode error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := db.FavoriteCreate(&gif, user); err != nil {
				fmt.Printf("Db error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(gif)
			return
		} else if r.Method == "GET" {
			id := r.FormValue("id")
			if id != "" {
				gif, err := db.FavoriteGet(id, user)
				if err != nil {
					fmt.Printf("Db error: %+v\n", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(gif)
				return
			}
			tag := r.FormValue("tag")
			if tag != "" {
				gifs, err := db.FavoriteListByTag(tag, user)
				if err != nil {
					fmt.Printf("Db error: %+v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(gifs)
				return
			}
			gifs, err := db.FavoriteList(user)
			if err != nil {
				fmt.Printf("Db error: %+v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gifs)
			return
		} else if r.Method == "DELETE" {
			id := r.FormValue("id")
			if err := db.FavoriteDelete(id, user); err != nil {
				fmt.Printf("Db error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

/* Adding/removing tags. Getting tag lists (full and by favorite). */
func tagsHandler(db Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUser(r)
		if user == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.ParseForm()
		if r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			var tag Tag
			if err := decoder.Decode(&tag); err != nil {
				fmt.Printf("Json decode error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := db.TagCreate(tag, user); err != nil {
				fmt.Printf("Db error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tag)
			return
		} else if r.Method == "GET" {
			favorite := r.FormValue("favorite")
			var tags []Tag
			var err error
			if favorite != "" {
				tags, err = db.FavoriteTagList(favorite, user)
			} else {
				tags, err = db.TagList(user)
			}
			if err != nil {
				fmt.Printf("Db error: %+v\n", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tags)
			return
		} else if r.Method == "DELETE" {
			tag := Tag{Favorite: r.FormValue("favorite"), Tag: r.FormValue("tag")}
			if err := db.TagDelete(tag, user); err != nil {
				fmt.Printf("Db error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

/* Handles register/login. Tells the client to set a cookie. */
func authHandler(db Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			user := r.FormValue("user")
			password := r.FormValue("password")
			sessionId, err := db.SessionCreate(user, password)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, err)
				return
			}
			expiration := time.Now().Add(24 * time.Hour)
			cookie := http.Cookie{Name: "sessionid", Value: sessionId, Expires: expiration, Path: "/"}
			http.SetCookie(w, &cookie)
			fmt.Fprintln(w, "Success")
			return
		} else if r.Method == "PUT" {
			r.ParseForm()
			user := r.FormValue("user")
			password := r.FormValue("password")
			if err := db.AccountCreate(user, password); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, err)
				fmt.Printf("AccountCreate error: %+v\n", err)
				return
			}
			sessionId, err := db.SessionCreate(user, password)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, err)
				return
			}
			expiration := time.Now().Add(24 * time.Hour)
			cookie := http.Cookie{Name: "sessionid", Value: sessionId, Expires: expiration, Path: "/"}
			http.SetCookie(w, &cookie)
			fmt.Fprintln(w, "Success")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

/* Sets up the db connection, the giphy client, and the http server. */
/* The handler chain looks like:
 * logger -> static -> cors -> auth -> router */
func main() {
	dbpath := "foo.db"
	rand.Seed(time.Now().UTC().UnixNano())
	db, err := NewSqliteDB(dbpath)
	if err != nil {
		log.Fatal("DB err: %+v\n", err)
	}
	fmt.Printf("Using database: %+v\n", dbpath)
	h := http.NewServeMux()
	var apiKey string
	if key, ok := os.LookupEnv("NOTGIPHY_API_KEY"); ok {
		apiKey = key
	} else {
		apiKey = "dc6zaTOxFJmzC"
	}
	fmt.Printf("Giphy api key: %v\n", apiKey)
	resultsPerPage := 25
	client := NewGifClient(apiKey, resultsPerPage)

	h.HandleFunc("/", rootHandler(db, client))
	h.HandleFunc("/api/gifs", gifsHandler(client))
	h.HandleFunc("/api/auth", authHandler(db))
	h.HandleFunc("/api/favorites", favoritesHandler(db))
	h.HandleFunc("/api/tags", tagsHandler(db))

	authed := authRequired(h, db)
	cors := openCORS(authed)
	static := staticHandler(cors)
	logged := requestLog(static)

	err = http.ListenAndServe(":9999", logged)
	if err != nil {
		log.Fatal(err)
	}
}
