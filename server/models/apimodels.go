package models

import (
	"encoding/json"
	"errors"
	"time"
)

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

type KUser struct {
	UserId int    `json:"userId"`
	DJName string `json:"username"`
}

type KGameProfileRaw struct {
	Stats struct {
		UserId  int `json:"userId"`
		Classes struct {
			Dan *string `json:"dan"`
		} `json:"classes"`
	} `json:"gameStats"`
}

type KGameProfile struct {
	UserId     int `json:"userId"`
	SPDanLevel int `json:"spDanLevel"`
}

func (k KGameProfileRaw) Flatten() KGameProfile {

	if k.Stats.Classes.Dan == nil {
		return KGameProfile{
			UserId:     k.Stats.UserId,
			SPDanLevel: 0,
		}
	}

	return KGameProfile{
		UserId:     k.Stats.UserId,
		SPDanLevel: KDanStrings[*k.Stats.Classes.Dan],
	}
}

type KScoreRaw struct {
	ScoreId string `json:"scoreID"`
	ChartId string `json:"chartID"`
	// unix timestamp - utc milliseconds
	TimeAchieved int64 `json:"timeAchieved"`
	UserId       int   `json:"userID"`
	ScoreData    struct {
		ExScore     int `json:"score"`
		EnumIndexes struct {
			Lamp int `json:"lamp"`
		} `json:"enumIndexes"`
		Optional struct {
			BP int `json:"bp"`
		} `json:"optional"`
		Lamp string `json:"lamp"`
	} `json:"scoreData"`
}

// need to correlate it *with* a chart to get the difficulty
type KScore struct {
	ScoreId string `json:"scoreID"`
	ChartId string `json:"chartID"`
	// unix timestamp - utc milliseconds
	TimeAchieved int64 `json:"timeAchieved"`
	UserId       int   `json:"userID"`
	ExScore      int   `json:"score"`
	Lamp         int   `json:"lamp"`
	BP           int   `json:"bp"`
}

func (k KScoreRaw) Flatten() KScore {
	return KScore{
		ScoreId:      k.ScoreId,
		ChartId:      k.ChartId,
		TimeAchieved: k.TimeAchieved,
		UserId:       k.UserId,
		ExScore:      k.ScoreData.ExScore,
		Lamp:         k.ScoreData.EnumIndexes.Lamp,
		BP:           k.ScoreData.Optional.BP,
	}
}

// this is the bare minimum of what we need to correlate a score with a bracket.
type KChart struct {
	ChartId    string         `json:"chartID"`
	Metadata   KChartMetadata `json:"data"`
	Difficulty string         `json:"difficulty"`
}

type KChartMetadata struct {
	SongIds KInGameID `json:"inGameID"`
}

type KInGameID struct {
	Values []int
}

func (i KInGameID) Last() int {
	if len(i.Values) == 0 {
		return 0
	}
	return i.Values[len(i.Values)-1]
}

func (i *KInGameID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.Values = nil
		return nil
	}

	var single int
	if err := json.Unmarshal(data, &single); err == nil {
		i.Values = []int{single}
		return nil
	}

	var many []int
	if err := json.Unmarshal(data, &many); err == nil {
		i.Values = many
		return nil
	}

	return errors.New("inGameID must be an int or []int")
}

type KChartScore struct {
	SongId     int
	UserId     int `json:"userID"`
	Difficulty string
	Timestamp  time.Time
	ExScore    int `json:"score"`
	Lamp       int `json:"lamp"`
	BP         int `json:"bp"`
}

func (k KChart) Flatten(score KScore) KChartScore {
	return KChartScore{
		SongId:     k.Metadata.SongIds.Last(), // always use newest song ID
		UserId:     score.UserId,
		Difficulty: k.Difficulty,
		Timestamp:  time.UnixMilli(score.TimeAchieved),
		ExScore:    score.ExScore,
		Lamp:       score.Lamp,
		BP:         score.BP,
	}
}

// K API is quite verbose and returns a lot of data - we can throw most of it away after deserializing it into this struct
// KScore looks something like this:
/*

```json
 "body": {
	"charts": [
		{
			"game": "iidx-sp",
			"chartID": "C19d35e109b6fb5d96ca",
			"legacyChartID": "fc7edc6bcfa701a261c89c999ddbba3e2195597b",
			"song": {
				"altTitles": [],
				"artist": "SLAKE",
				"data": {
					"genre": "BIG BEAT",
					"displayVersion": "1"
				},
				"id": "S19d35e0aee33127a694",
				"searchTerms": [],
				"title": "GAMBOL"
			},
			"level": "2",
			"levelNum": 2,
			"isPrimary": true,
			"difficulty": "HYPER",
			"data": {
				"inGameID": 1006,
				"2dxtraSet": null,
				"notecount": 137,
				"hashSHA256": null,
				"worldRecord": null,
				"kaidenAverage": null,
				"bpiCoefficient": null
			},
			"versions": []
		}
	],
	"songs": [
		{
			"id": "S19d35e0aee33127a694",
			"title": "GAMBOL",
			"artist": "SLAKE",
			"searchTerms": [],
			"altTitles": [],
			"data": {
				"genre": "BIG BEAT",
				"displayVersion": "1"
			}
		}
	],
	"scores": [
		{
			"service": "FLO",
			"game": "iidx-sp",
			"userID": 6290,
			"scoreData": {
				"enumIndexes": {
					"lamp": 5,
					"grade": 6
				},
				"optional": {
					"enumIndexes": {},
					"fast": null,
					"slow": null,
					"bp": 7
				},
				"judgements": {},
				"score": 1951,
				"lamp": "HARD CLEAR",
				"percent": 85.4951796669588,
				"grade": "AA"
			},
			"scoreMeta": {},
			"calculatedData": {
				"BPI": null,
				"ktLampRating": 10,
				"ktLampRatingHC": 10,
				"ktLampRatingEXHC": 0
			},
			"timeAchieved": 1783276458000,
			"songID": "S19d35e0b57fb77b9b12",
			"chartID": "C19d35e1105736f994ca",
			"isPrimary": true,
			"highlight": false,
			"comment": null,
			"timeAdded": 1783277678248,
			"scoreID": "Tba23276782760ac70baf2da202745fbefd2423edbd9bd17d85f0af2a9929c3db",
			"sessionID": "Q3e49fd54763b0602a8b71a1af04b97e5af354f5d",
			"importType": "api/flo-iidx"
		}
	]
	...
```

*/
