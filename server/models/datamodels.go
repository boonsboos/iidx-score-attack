package models

import (
	"database/sql"
	"time"
)

// a player
type Player struct {
	ID           uint           `json:"id" gorm:"index,primarykey"`
	GameID       int            `json:"game_id" gorm:"unique"` // the xxxx-xxxx id in game, since DJName is not unique.
	DJName       string         `json:"name"`
	DanLevel     int            `json:"dan_level"`                            // 7k = 0, 1d = 7, ... 10d = 16, chuuden = 17, kaiden = 18
	RefreshToken sql.NullString `json:"refresh_token" gorm:"index,size:1024"` // can only be nil or "" if the user revoked access to us and we already tried to make a refresh request that failed.
}

// the pool of charts that are currently active for a score attack event
type ChartPool struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Title       string    `json:"title"`
	ActiveFrom  time.Time `json:"active_from"`
	ActiveUntil time.Time `json:"active_to"`
}

// a chart that is part of a score attack event
type BracketChart struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	PoolID      uint      `json:"pool_id"`
	Pool        ChartPool `json:"pool"`
	ChartID     uint      `json:"chart_id"`
	Chart       Chart     `json:"chart"`
	BracketType string    `json:"bracket_type"` // "upper" or "lower"
	ChartType   string    `json:"chart_type"`   // "normal" or "boss"
}

// a chart in game
type Chart struct {
	ID         uint          `json:"id" gorm:"primarykey"`
	SongId     uint          `json:"iidx_song_id"`
	Song       Song          `json:"song"`
	Difficulty string        `json:"level"`      // B, N, H, A or L
	Level      int           `json:"difficulty"` // 1 to 12
	MaxScore   sql.NullInt32 `json:"max_score"`
}

// a song in game
type Song struct {
	ID            uint         `json:"id" gorm:"primarykey"`
	Name          string       `json:"name"`
	NameLatinized string       `json:"name_lat"`
	Artist        string       `json:"artist"`
	Charts        []Chart      `json:"charts"`
	VersionId     uint         `json:"version_id"`
	Version       *GameVersion `json:"version"`
}

// versions of the game.
// shown next to charts to help users find it in the folder
type GameVersion struct {
	ID   uint   `json:"id" gorm:"primarykey"`
	Name string `json:"name"`
}

// a score set by a user on a chart in the bracket.
// **note**: the reason it's tied to BracketChart and not Chart is because the same chart can be in multiple brackets, and we want to keep scores separate for each bracket.
type Score struct {
	ID             uint         `json:"id" gorm:"primarykey"`
	PlayerID       uint         `json:"player_id" gorm:"index:idx_player_chart,unique"`
	Player         Player       `json:"player"`
	BracketChartID uint         `json:"chart_id" gorm:"index:idx_player_chart,unique"`
	BracketChart   BracketChart `json:"chart"`
	Ex             int          `json:"ex"`
	Misscount      int          `json:"misscount"`
	Lamp           int          `json:"lamp"`
	Timestamp      time.Time    `json:"timestamp"`
}

func (pool *ChartPool) IsActive() bool {
	now := time.Now()
	return now.After(pool.ActiveFrom) && now.Before(pool.ActiveUntil)
}

func (pool *ChartPool) IsValid() bool {
	return pool.ActiveFrom.Before(pool.ActiveUntil)
}

var LampStrings = map[int]string{
	0: "NO PLAY",
	1: "FAILED",
	2: "ASSIST CLEAR",
	3: "EASY CLEAR",
	4: "CLEAR",
	5: "HARD CLEAR",
	6: "EX-HARD CLEAR",
	7: "FULL COMBO",
}

var DanStrings = map[int]string{
	0:  "七級",
	1:  "六級",
	2:  "五級",
	3:  "四級",
	4:  "三級",
	5:  "二級",
	6:  "一級",
	7:  "初段",
	8:  "二段",
	9:  "三段",
	10: "四段",
	11: "五段",
	12: "六段",
	13: "七段",
	14: "八段",
	15: "九段",
	16: "十段",
	17: "中伝",
	18: "皆伝",
}

var DanStringsLatin = map[int]string{
	0:  "7th kyu",
	1:  "6th kyu",
	2:  "5th kyu",
	3:  "4th kyu",
	4:  "3rd kyu",
	5:  "2nd kyu",
	6:  "1st kyu",
	7:  "1st dan",
	8:  "2nd dan",
	9:  "3rd dan",
	10: "4th dan",
	11: "5th dan",
	12: "6th dan",
	13: "7th dan",
	14: "8th dan",
	15: "9th dan",
	16: "10th dan",
	17: "Chuuden",
	18: "Kaiden",
}
