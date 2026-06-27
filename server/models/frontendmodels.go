package models

type FrontendBracketChart struct {
	Title          string `json:"title"`
	TitleLatinized string `json:"title_lat"` // for use in tooltips since titles can be in japanese
	Artist         string `json:"artist"`
	ChartLevel     string `json:"level"`      // SPA9, SPH7 etc.
	Version        string `json:"version"`    // 1st, 2nd, 3rd etc.
	VersionId      uint   `json:"version_id"` // for styling the version in the table
	ChartType      string `json:"chart_type"` // "normal" or "boss"
}

type FrontendPlayerScore struct {
	ChartId uint `json:"chart_id"`
	// if 0, unplayed, don't show other data
	ExScore int `json:"ex_score"`
	// % of max score
	ScoreRate string  `json:"score_rate"`
	Misscount int     `json:"misscount"`
	Lamp      int     `json:"lamp"`
	Rating    float64 `json:"rating"`
	Timestamp string  `json:"timestamp"`
}

type ScorePageBracketChart struct {
	Title          string `json:"title"`
	TitleLatinized string `json:"title_lat"`  // for use in tooltips since titles can be in japanese
	ChartLevel     string `json:"level"`      // SPA9, SPH7 etc.
	ChartType      string `json:"chart_type"` // "normal" or "boss"
}

type ScorePagePlayerScore struct {
	Player Player  `json:"player"`
	Rating float64 `json:"rating"`
	// assume that the scores are sorted by chart.
	Scores []FrontendPlayerScore `json:"scores"`
}
