package firebase_client

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/v4/messaging"
)

// maxPushTargets is the limit of devices of one multicast message in FCM.
const maxPushTargets = 500

var _ PushSender = (*FirebaseClient)(nil)

// Send pushes msg to the devices in chunks of maxPushTargets. The frontend registers an FCM registration token
// (FirebaseMessaging.getToken) or a Firebase Installation ID (FID registration); both are accepted (DP-050).
func (c *FirebaseClient) Send(ctx context.Context, tokens []string, msg PushMessage) (PushResult, error) {
	var result PushResult
	for _, chunk := range chunkTargets(tokens, maxPushTargets) {
		registrationTokens, fids := splitTargets(chunk)
		resp, err := c.messaging.SendEachForMulticast(ctx, &messaging.MulticastMessage{
			// The SDK deprecates registration tokens in favour of FIDs, but Flutter apps still get tokens and FCM keeps
			// accepting them. TODO(DP-050): drop them once the app registers FIDs.
			Tokens:       registrationTokens, //nolint:staticcheck // SA1019: registration tokens are still supported
			Fids:         fids,
			Notification: &messaging.Notification{Title: msg.Title, Body: msg.Body},
			Data:         msg.Data,
			Android:      &messaging.AndroidConfig{Priority: "high"},
			APNS:         &messaging.APNSConfig{Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default"}}},
		})
		if err != nil {
			return result, fmt.Errorf("send push notification: %w", err)
		}
		// The responses follow the order of the targets of the message: the tokens and then the FIDs.
		targets := append(append([]string{}, registrationTokens...), fids...)
		errs := make([]error, len(resp.Responses))
		for i, r := range resp.Responses {
			if !r.Success {
				errs[i] = r.Error
			}
		}
		addResults(&result, targets, errs, isInvalidTarget)
	}
	return result, nil
}

// splitTargets separates the FCM registration tokens, which have the form <FID>:<suffix>, from the bare FIDs.
func splitTargets(targets []string) (registrationTokens, fids []string) {
	for _, t := range targets {
		if strings.Contains(t, ":") {
			registrationTokens = append(registrationTokens, t)
		} else {
			fids = append(fids, t)
		}
	}
	return registrationTokens, fids
}

// chunkTargets splits targets in slices of at most size elements.
func chunkTargets(targets []string, size int) [][]string {
	var chunks [][]string
	for len(targets) > size {
		chunks = append(chunks, targets[:size])
		targets = targets[size:]
	}
	if len(targets) > 0 {
		chunks = append(chunks, targets)
	}
	return chunks
}

// addResults adds to result the outcome of each target: errs[i] is nil when targets[i] received the message.
func addResults(result *PushResult, targets []string, errs []error, invalid func(error) bool) {
	for i, err := range errs {
		switch {
		case err == nil:
			result.SuccessCount++
		case invalid(err):
			result.InvalidTokens = append(result.InvalidTokens, targets[i])
		default:
			result.TransientFailures++
		}
	}
}

// isInvalidTarget reports whether FCM rejected the device for good, so its token must be forgotten.
func isInvalidTarget(err error) bool {
	return messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) || messaging.IsSenderIDMismatch(err)
}
