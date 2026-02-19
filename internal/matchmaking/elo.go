package matchmaking

import "math"

const (
	DefaultElo = 1200
	KFactor    = 32
)

// Band returns the tier band (1-3) for a given Elo rating.
func Band(elo int) int {
	switch {
	case elo < 1000:
		return 1
	case elo < 1400:
		return 2
	default:
		return 3
	}
}

// ExpectedScore returns the expected outcome probability for playerA vs playerB.
func ExpectedScore(eloA, eloB int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(eloB-eloA)/400.0))
}

// NewRating calculates the updated Elo rating.
// actual = 1.0 for win, 0.0 for loss, 0.5 for draw.
func NewRating(currentElo int, opponentElo int, actual float64) int {
	expected := ExpectedScore(currentElo, opponentElo)
	return currentElo + int(math.Round(KFactor*(actual-expected)))
}

// UpdateRatingsAfterRoom updates Elo for all players after a round.
// The winner gets a "win" against the average Elo of all opponents.
// Losers get a "loss" against the winner's Elo.
func UpdateRatingsAfterRoom(playerElos map[int64]int, winnerID int64) map[int64]int {
	results := make(map[int64]int, len(playerElos))

	if winnerID == 0 || len(playerElos) < 2 {
		for id, elo := range playerElos {
			results[id] = elo
		}
		return results
	}

	avgElo := 0
	for _, e := range playerElos {
		avgElo += e
	}
	avgElo /= len(playerElos)

	winnerElo := playerElos[winnerID]

	for id, elo := range playerElos {
		if id == winnerID {
			results[id] = NewRating(elo, avgElo, 1.0)
		} else {
			results[id] = NewRating(elo, winnerElo, 0.0)
		}
	}

	return results
}
