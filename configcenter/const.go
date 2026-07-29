package configcenter

import (
	"errors"
	"time"
)

const (
	// KeyBaseURL is the Config Center HTTP base URL for SDK clients.
	KeyBaseURL = "XX_WG_CONFIG_CENTER_URL"
	// KeyToken is the optional SDK bearer token (CONFIG_CENTER_SDK_TOKEN on server).
	KeyToken = "XX_WG_CONFIG_CENTER_TOKEN"
	// KeyInstanceID optionally identifies this process in heartbeats.
	KeyInstanceID = "XX_WG_INSTANCE_ID"

	// DefaultPollInterval matches Config Center server convention.
	DefaultPollInterval = 5 * time.Second
)

var (
	ErrNotFound     = errors.New("configcenter: bundle not found")
	ErrUnauthorized = errors.New("configcenter: unauthorized")
)
