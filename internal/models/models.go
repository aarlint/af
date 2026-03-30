package models

type President struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Term      int    `json:"term"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type Action struct {
	ID        int               `json:"id"`
	President string            `json:"president"`
	EO        string            `json:"eo"`
	Title     string            `json:"title"`
	Date      string            `json:"date"`
	Category  string            `json:"category"`
	URL       string            `json:"url"`
	Impacts   map[string]int    `json:"impacts,omitempty"`
	Reasons   map[string]string `json:"reasons,omitempty"`
}

type Impact struct {
	ActionID int    `json:"action_id"`
	Country  string `json:"country"`
	Score    int    `json:"score"`
	Reason   string `json:"reason"`
}

type CountryScore struct {
	Country    string `json:"country"`
	Total      int    `json:"total"`
	Positive   int    `json:"positive"`
	Negative   int    `json:"negative"`
	Cumulative []int  `json:"cumulative"`
}

type ActionsResponse struct {
	Actions  []Action `json:"actions"`
	Total    int      `json:"total"`
	Filtered int      `json:"filtered"`
}

type ScoresResponse struct {
	Countries  []string                `json:"countries"`
	Scores     map[string]CountryScore `json:"scores"`
	Presidents []President             `json:"presidents"`
}

// SeedAction is the format used in seed JSON files.
type SeedAction struct {
	EO       string            `json:"eo"`
	Title    string            `json:"title"`
	Date     string            `json:"date"`
	Category string            `json:"category"`
	URL      string            `json:"url"`
	Impacts  map[string]int    `json:"impacts"`
	Reasons  map[string]string `json:"reasons"`
}

type SeedFile struct {
	President string       `json:"president"`
	Actions   []SeedAction `json:"actions"`
}
