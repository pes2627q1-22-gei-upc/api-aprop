// Package language_model defines the languages of the application.
package language_model

// Supported languages (ISO 639-1). The events from the open data and the ones created by users are in Catalan.
const (
	Catalan = "ca"
	Spanish = "es"
	English = "en"

	// Default is the language of the original event texts and of the users that have not chosen one.
	Default = Catalan
)

// Valid reports whether lang is one of the supported languages.
func Valid(lang string) bool {
	return lang == Catalan || lang == Spanish || lang == English
}
