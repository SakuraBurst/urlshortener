package types

type URLShorter struct {
	ID          string `json:"-"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
