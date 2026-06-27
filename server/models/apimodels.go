package models

import "time"

// minimal fields needed to correlate a score with a bracket.
// at this point in the flow we already know which user we're fetching data for
type FScore struct {
	SongId int `json:"music_id"`
	// Used to differentiate between DP and SP
	PlayStyle string `json:"play_style"`
	// Used to differentiate between charts of the same song and playstyle
	Difficulty string    `json:"difficulty"`
	ExScore    int       `json:"ex_score"`
	MissCount  int       `json:"miss_count"`
	Lamp       int       `json:"lamp"`
	Timestamp  time.Time `json:"timestamp"`
}

type FPlayer struct {
	DJName     string `json:"dj_name"`
	GameID     int    `json:"iidx_id"`
	SPDanLevel int    `json:"sp"`
}
