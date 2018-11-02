package main

import (
	"fmt"
	"math/rand"
	"sync"
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

type memorydb struct {
	accounts map[string]string
	sessions map[string]string
	m        sync.Mutex
}

func (m *memorydb) AccountCreate(user, password string) (error) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.accounts[user]; ok {
		return fmt.Errorf("User already exists")
	}
	if password == "" {
		return fmt.Errorf("Invalid password")
	}
	m.accounts[user] = password
	return nil
}

func RandomCookie() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int63n(0xffffffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffff),
		rand.Int63n(0xffffffffffff))
}

func (m *memorydb) SessionCreate(user, password string) (string, error) {
	m.m.Lock()
	defer m.m.Unlock()
	foundUser := false
	for k, v := range m.accounts {
		if k == user && v == password {
			foundUser = true
		}
	}
	if !foundUser {
		return "", fmt.Errorf("Invalid user or password")
	}
	for k, v := range m.sessions {
		if v == user {
			delete(m.sessions, k)
			break
		}
	}
	cookie := RandomCookie()
	m.sessions[cookie] = user
	return cookie, nil
}

func (m *memorydb) SessionGet(cookie string) (string, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if val, ok := m.sessions[cookie]; ok {
		return val, nil
	}
	return "", fmt.Errorf("Invalid cookie")
}
func (m *memorydb) FavoriteCreate(user string, gif *Gif) error {
	return  nil
}
func (m *memorydb) FavoriteDelete(id, user string) error {
	return nil
}
func (m *memorydb) FavoriteList(user string, offset int) ([]Gif, error) {
	return nil, nil
}
func (m *memorydb) FavoriteGet(id, user string) (*Gif, error) {
	return nil, nil
}

/*
func NewMemoryDB() Db {
	return &memorydb{
		accounts: make(map[string]string),
		sessions: make(map[string]string),
	}
}
*/
