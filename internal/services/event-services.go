package services

import (
	. "api-aprop/internal/repositories"
	. "api-aprop/internal/models"
)

func ServeGetEvent(id int) (Event, error) {
	ev, err := Events.Get(id)
	if err==nil {
		return ev, err
	}
	return Event{}, err
}

func ServeListEvents() ([]Event) {
	return Events.List()
}