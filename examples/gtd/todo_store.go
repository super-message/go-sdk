package main

import (
	"sync"
)

type Todo struct {
	ID     int    `json:"id"`
	UserID string `json:"uid"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
}

type TodoStore struct {
	_id   int
	list  []*Todo
	mutex sync.RWMutex
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		list: make([]*Todo, 0, 8),
	}
}

func (s *TodoStore) ListTodo(openID string) []*Todo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var list = make([]*Todo, 0, 8)
	for i, r := range s.list {
		if r.UserID == openID {
			list = append(list, s.list[i])
		}
	}

	return list
}

func (s *TodoStore) AddTodo(todo *Todo) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s._id++
	todo.ID = s._id
	s.list = append(s.list, todo)
}

func (s *TodoStore) GetTodo(id int) *Todo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, r := range s.list {
		if r.ID == id {
			return r
		}
	}

	return nil
}

func (s *TodoStore) DeleteTodo(id int, openID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, r := range s.list {
		if r.ID == id {
			if r.UserID == openID {
				s.list = append(s.list[:i], s.list[i+1:]...)
			}

			break
		}
	}
}
