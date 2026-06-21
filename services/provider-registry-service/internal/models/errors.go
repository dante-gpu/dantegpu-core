package models

import "errors"

// Predefined errors for the provider store and handlers.
var (
	ErrProviderNotFound      = errors.New("provider not found")
	ErrProviderAlreadyExists = errors.New("provider already exists with this ID")
	ErrInvalidProviderData   = errors.New("invalid provider data provided")
	// Add more specific errors as needed
)
