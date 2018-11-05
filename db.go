package main

import (
	"fmt"
	"math/rand"
)

type Db interface {
	AccountCreate(user, password string) error
	SessionCreate(user, password string) (string, error)
	SessionGet(cookie string) (string, error)
	FavoriteCreate(gif *Gif, user string) error
	FavoriteDelete(id, user string) error
	FavoriteList(user string) ([]Gif, error)
	FavoriteGet(id, user string) (*Gif, error)
	TagCreate(tag Tag, user string) error
	TagDelete(tag Tag, user string) error
	TagList(user string) ([]Tag, error)
	FavoriteTagList(favorite, user string) ([]Tag, error)
	FavoriteListByTag(tag, user string) ([]Gif, error)
}

func RandomCookie() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int63n(0xffffffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffffffffffff))
}
