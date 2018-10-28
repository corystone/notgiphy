package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

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
		fmt.Printf("Searching for %v\n", q)
		gifs, err := client.Search(q)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal Server Error")
			fmt.Printf("Error: %v", err)
			return
		}
		for _, gif := range gifs {
			fmt.Printf("GIF: %v\n", gif)
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gifs)
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
				return
			}
			fmt.Fprintln(w, "Account created")
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
	client := NewGifClient(apiKey)

	h.HandleFunc("/", rootHandler(db, client))
	h.HandleFunc("/auth", authHandler(db))
	h.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, you hit foo!")
	})

	h.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, you hit bar!")
	})

	authed := authRequired(h, db)

	err := http.ListenAndServe(":9999", authed)
	log.Fatal(err)
}

