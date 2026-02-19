package room

import "time"

type RoomType string

const (
	RoomAlpha RoomType = "alpha"
	RoomBlitz RoomType = "blitz"
)

type RoomState int

const (
	StateWaiting RoomState = iota
	StateActive
	StateSurvival
	StateFinished
)

func (s RoomState) String() string {
	switch s {
	case StateWaiting:
		return "waiting"
	case StateActive:
		return "active"
	case StateSurvival:
		return "survival"
	case StateFinished:
		return "finished"
	default:
		return "unknown"
	}
}

type TierConfig struct {
	Tier          int
	EntryCost     int64
	MinPlayers    int
	MaxPlayers    int
	PulseWindow   time.Duration // time a player has to pulse before elimination
	BaseExtension time.Duration // base timer extension per pulse
	SurvivalTime  time.Duration // total survival phase duration
	PrestigeMult  float64
}

var Tiers = map[int]TierConfig{
	1: {
		Tier:          1,
		EntryCost:     5,
		MinPlayers:    3,
		MaxPlayers:    20,
		PulseWindow:   5 * time.Second,
		BaseExtension: 3 * time.Second,
		SurvivalTime:  120 * time.Second,
		PrestigeMult:  1.0,
	},
	2: {
		Tier:          2,
		EntryCost:     20,
		MinPlayers:    5,
		MaxPlayers:    30,
		PulseWindow:   4 * time.Second,
		BaseExtension: 2500 * time.Millisecond,
		SurvivalTime:  150 * time.Second,
		PrestigeMult:  1.5,
	},
	3: {
		Tier:          3,
		EntryCost:     100,
		MinPlayers:    5,
		MaxPlayers:    50,
		PulseWindow:   3 * time.Second,
		BaseExtension: 2 * time.Second,
		SurvivalTime:  180 * time.Second,
		PrestigeMult:  2.0,
	},
}

type PlayerState struct {
	ID           int64
	Username     string
	Alive        bool
	PulseCount   int
	StarsSpent   int64
	JoinedAt     time.Time
	LastPulseAt  time.Time
	EliminatedAt *time.Time
}
