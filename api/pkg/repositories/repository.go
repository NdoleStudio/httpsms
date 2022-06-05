package repositories

// IndexParams parameters for indexing a database table
type IndexParams struct {
	Skip  int    `json:"skip"`
	Query string `json:"query"`
	Limit int    `json:"take"`
}
