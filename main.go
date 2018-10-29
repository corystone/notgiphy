package main

import (
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
		fmt.Printf("User (%v) has a cookie", user)
		next.ServeHTTP(w, r)
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
		page := 1
		p := r.FormValue("p")
		if p != "" {
			var err error
			page, err = strconv.Atoi(p)
			if err != nil {
				page = 1
			}
		}
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

func gifHandler(client *GifClient) http.HandlerFunc {
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
			_, err := db.AccountCreate(user, password)
			if err != nil {
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
	db := NewDB()
	h := http.NewServeMux()
	// FIXME. Config or ENV or something. This is the "public beta key"
	apiKey := "dc6zaTOxFJmzC"
	resultsPerPage := 25
	client := NewGifClient(apiKey, resultsPerPage)

	h.HandleFunc("/", rootHandler(db, client))
	h.HandleFunc("/gif", gifHandler(client))
	h.HandleFunc("/auth", authHandler(db))
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

	err := http.ListenAndServe(":9999", cors)
	log.Fatal(err)
}

