package types

import "net/url"

type URLShorter struct {
	ShortURL    string   `json:"short_url"`
	OriginalURL *url.URL `json:"original_url"`
}
