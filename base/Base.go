package base

// AudiobookTrack describes a track in the audiobook.
type AudiobookTrack struct {
	Ord      int    `json:"ord"`
	Title    string `json:"title"`
	Filename string `json:"filename"`
}

// Audiobook describes an audiobook.
type Audiobook struct {
	ID     			string 			`json:"id"`
	Artist			string			`json:"Artist"`
	Title  			string 			`json:"title"`
	ArchiveFile string 			`json:"archiveFile"`
	Tracks []AudiobookTrack `json:"tracks"`
}

// AudiobookDirectory describes an audiobook directory.
type AudiobookDirectory struct {
	ID      string      `json:"id"`
	BaseURL string      `json:"baseURL"`
	Books   []Audiobook `json:"books"`
}
