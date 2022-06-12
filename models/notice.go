package models

import "sync"

type AutoInc struct {
	sync.Mutex
	Id int
}

func (a *AutoInc) ID() (id int) {
	a.Lock()
	defer a.Unlock()

	id = a.Id
	a.Id++
	return
}

type Notice struct {
	ID  int    `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
	Date  string `json:"date"`
}