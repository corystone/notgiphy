package main

import (
	"fmt"
	"math/rand"
	"sync"
)

type Db interface {
	AccountCreate(user, password string) (int, error)
	SessionCreate(user, password string) (string, error)
	SessionGet(cookie string) (string, error)
}

type memorydb struct {
	accounts map[string]string
	sessions map[string]string
	m        sync.Mutex
}

func (m *memorydb) AccountCreate(user, password string) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.accounts[user]; ok {
		return -1, fmt.Errorf("User already exists")
	}
	if password == "" {
		return -1, fmt.Errorf("Invalid password")
	}
	m.accounts[user] = password
	return 1, nil
}

func randomCookie() string {
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
	cookie := randomCookie()
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

func NewDB() Db {
	return &memorydb{
		accounts: make(map[string]string),
		sessions: make(map[string]string),
	}
}
