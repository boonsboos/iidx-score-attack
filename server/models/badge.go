package models

var Badges = map[int]string{
	0: "OG",
	1: "August 2026 Hidden Theme",
	2: "September 2026 Hidden Theme",
	3: "TBD",
}

var BadgeIcons = map[int]string{
	0: `✨`,
	1: `🪙`,
	2: `💽`,
	3: `❓`,
}

func (player *Player) AssignBadge(badgeIndex int) {
	if badgeIndex < 0 || badgeIndex >= len(Badges) {
		return
	}

	// resize to fit
	playerBadges := []byte(player.Badge)

	if len(playerBadges) < len(Badges) {
		expanded := make([]byte, len(Badges))
		copy(expanded, playerBadges)
		for i := len(playerBadges); i < len(expanded); i++ {
			expanded[i] = '0'
		}
		playerBadges = expanded
	}

	// assign
	playerBadges[badgeIndex] = '1'
	player.Badge = string(playerBadges)
}
