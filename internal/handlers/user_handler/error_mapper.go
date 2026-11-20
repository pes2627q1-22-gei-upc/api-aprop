package user_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/user_service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user_service.ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, user_service.ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, user_service.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "forbidden: profile plans are private")
	case errors.Is(err, user_service.ErrUsernameTaken):
		response.WriteError(w, http.StatusConflict, "username already taken")
	case errors.Is(err, user_service.ErrInvalidInput), errors.Is(err, user_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrRequestAlreadyExists):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrIsBlocked):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrAlreadyFriends):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrRequestNotFound):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrNotFriends):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrSameUser):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user_service.ErrAlreadyBlocked):
		response.WriteError(w, http.StatusConflict, "user already blocked")
	case errors.Is(err, user_service.ErrNotBlocked):
		response.WriteError(w, http.StatusNotFound, "user is not blocked")
	case errors.Is(err, user_service.ErrStorageUnavailable):
		log.Printf("upload avatar: %v", err)
		response.WriteError(w, http.StatusServiceUnavailable, "avatar storage is not available")
	case errors.Is(err, user_service.ErrStorageFailed):
		log.Printf("upload avatar: %v", err)
		response.WriteError(w, http.StatusBadGateway, "could not store the profile picture")
	case errors.Is(err, user_service.ErrAuthProvider):
		log.Printf("delete account: %v", err)
		response.WriteError(w, http.StatusBadGateway, "could not delete the authentication account")
	default:
		log.Printf("user handler: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
