package json

//easyjson:json
type Result struct {
	Result string `json:"result"`
}

//easyjson:json
type URL struct {
	URL string `json:"url"`
}

//easyjson:json
type DbRow struct {
	Id          int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
