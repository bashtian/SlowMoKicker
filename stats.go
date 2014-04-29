package main

import (
	"fmt"
	"time"
)

type Stats struct {
	Team1        int64
	Team2        int64
	LastGoalTime time.Time
	LastGoalTeam int64
}

func (s *Stats) TextBytes() []byte {
	var text string
	if s.Team1 == 0 && s.Team2 == 0 {
		text = fmt.Sprintf("Game starts<br>0:0")
	} else if stats.IsFinshed() {
		text = fmt.Sprintf("Team %v Won<br>%v:%v", s.LastGoalTeam, s.Team1, s.Team2)
	} else {
		text = fmt.Sprintf("%v:%v<br>last goal from team %v", s.Team1, s.Team2, s.LastGoalTeam)
	}
	return []byte(text)
}

func (s *Stats) IsFinshed() bool {
	return s.Team1 >= maxPoints || s.Team2 >= maxPoints
}

func (s *Stats) ResetLastGoal() {
	switch s.LastGoalTeam {
	case goalTeam1:
		s.Team1--
	case goalTeam2:
		s.Team2--
	}
	s.LastGoalTeam = 0
}

func (s *Stats) Restart() {
	s.Team1, s.Team2 = 0, 0
	s.LastGoalTeam = 0
}
