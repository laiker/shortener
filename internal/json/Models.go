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
type DBRow struct {
	ID            int    `json:"uuid,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	ShortURL      string `json:"short_url,omitempty"`
	OriginalURL   string `json:"original_url,omitempty"`
	UserID        string `json:"user_id,omitempty"`
}

//easyjson:json
type BatchURLSlice []DBRow
