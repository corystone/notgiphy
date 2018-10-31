package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func openCORS(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		next.ServeHTTP(w, r)
	}
}

func authRequired(next http.Handler, db Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/auth" {
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
		fmt.Printf("User (%v) has a cookie\n", user)
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func rootHandler(db Db, client *GifClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "NO SOUP FOR YOU")
			return
		}
		r.ParseForm()
		q := r.FormValue("q")
		if q == "" {
			fmt.Fprintln(w, "FIXME. YOU ARE GETTING THE index.html page")
			return
		}
		page := getPage(r.FormValue("p"))
		fmt.Printf("Searching for %v, page: %v\n", q, page)
		gifs, err := client.Search(q, page)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal Server Error")
			fmt.Printf("Error: %v", err)
			return
		}
		for _, gif := range gifs {
			fmt.Printf("GIF: %v\n", gif)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gifs)
	}
}

func gifsHandler(client *GifClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintln(w, "Only GETS please")
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "BAD ID")
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
	fmt.Printf("No USER for authed call! (Shouldn't happen...)\n")
	return ""
}

func getPage(p string) int {
	page, err := strconv.Atoi(p)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

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
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, "Created")
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
			page := getPage(r.FormValue("p"))
			gifs, err := db.FavoriteList(user, (page - 1) * 25)
			if err != nil {
				fmt.Printf("Db error: %+v\n", err)
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
			fmt.Fprintln(w, "Created")
			return
		} else if r.Method == "GET" {
			favorite := r.FormValue("favorite")
			var tags []Tag
			var err error
			fmt.Printf("GET TAGS, favorite: %+v\n", favorite)
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
			decoder := json.NewDecoder(r.Body)
			var tag Tag
			if err := decoder.Decode(&tag); err != nil {
				fmt.Printf("Json decode error: %+v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
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
			cookie := http.Cookie{Name: "sessionid", Value: sessionId, Expires: expiration}
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
			cookie := http.Cookie{Name: "sessionid", Value: sessionId, Expires: expiration}
			http.SetCookie(w, &cookie)
			fmt.Fprintln(w, "Success")
		} else if r.Method == "OPTIONS" {
			fmt.Fprintln(w, "SURE, HAVE SOME OPTIONS")
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "NO SOUP FOR YOU")
			return
		}
	}
}

func main() {
	//db := NewMemoryDB()
	fmt.Println("Get a db!")
	db, err := NewSqliteDB("foo.db")
	if err != nil {
		log.Fatal("DB err: %+v\n", err)
	}
	fmt.Println("GOT a db!")
	h := http.NewServeMux()
	// FIXME. Config or ENV or something. This is the "public beta key"
	apiKey := "dc6zaTOxFJmzC"
	resultsPerPage := 25
	client := NewGifClient(apiKey, resultsPerPage)

	h.HandleFunc("/", rootHandler(db, client))
	h.HandleFunc("/gifs", gifsHandler(client))
	h.HandleFunc("/auth", authHandler(db))
	h.HandleFunc("/favorites", favoritesHandler(db))
	h.HandleFunc("/tags", tagsHandler(db))
	h.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, you hit foo!")
	})

	h.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, you hit bar!")
	})

	// FIXME. The auth off switch.
	authed := authRequired(h, db)
	cors := openCORS(authed)
	//cors := openCORS(h)

	err = http.ListenAndServe(":9999", cors)
	if err != nil {
		log.Fatal(err)
	}
}

