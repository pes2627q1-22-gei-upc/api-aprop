package services

import (
	. "api-aprop/internal/repositories"
	. "api-aprop/internal/models"
)

var Events EventStore = NewMemoryStore()

func ServeGetEvent(id int) (Event, error) {
	ev, err := Events.Get(id)
	if err==nil {
		return ev, err
	}
	return Event{}, err
}