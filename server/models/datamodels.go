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
	DanLevel     int            `json:"dan_level"`                  // 7k = 0, 1d = 7, ... 10d = 16, chuuden = 17, kaiden = 18
	RefreshToken sql.NullString `json:"refresh_token" gorm:"index"` // can only be nil or "" if the user revoked access to us and we already tried to make a refresh request that failed.
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
