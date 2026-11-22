package push_service

import (
	"errors"
	"fmt"

	"api-aprop/internal/models/push_model"
	"api-aprop/internal/models/user_model"
)

var (
	ErrInvalidUserID   = user_model.ErrInvalidID
	ErrInvalidToken    = fmt.Errorf("token must have between 1 and %d characters", push_model.MaxTokenLength)
	ErrInvalidPlatform = errors.New("platform must be android, ios or web")
	// ErrPushDisabled is returned by DispatchPending when there is no FCM client (no Firebase credentials).
	ErrPushDisabled = errors.New("push notifications are disabled: no Firebase credentials")
)
