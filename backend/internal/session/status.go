package session

import (
	"fmt"
	"time"
)

type SessionStatus string

const (
	StatusPlanning     SessionStatus = "planning"
	StatusScheduled    SessionStatus = "scheduled"
	StatusInProcessing SessionStatus = "in_processing"
	StatusHumanReview  SessionStatus = "human_review"
	StatusCompleted    SessionStatus = "completed"
)

type StatusTransition struct {
	FromStatus SessionStatus
	ToStatus   SessionStatus
	Timestamp  int64
	Reason     string
}

func NewStatusTransition(from, to SessionStatus, reason string) StatusTransition {
	return StatusTransition{
		FromStatus: from,
		ToStatus:   to,
		Timestamp:  time.Now().Unix(),
		Reason:     reason,
	}
}

func (t StatusTransition) String() string {
	return fmt.Sprintf("[%s] %s -> %s: %s",
		time.Unix(t.Timestamp, 0).Format("2006-01-02 15:04:05"),
		t.FromStatus,
		t.ToStatus,
		t.Reason)
}

func StatusDisplayName(status SessionStatus) string {
	switch status {
	case StatusPlanning:
		return "规划中"
	case StatusScheduled:
		return "排期中"
	case StatusInProcessing:
		return "处理中"
	case StatusHumanReview:
		return "人工介入"
	case StatusCompleted:
		return "已完成"
	default:
		return string(status)
	}
}

func StatusColor(status SessionStatus) string {
	switch status {
	case StatusPlanning:
		return "\033[34m"
	case StatusScheduled:
		return "\033[33m"
	case StatusInProcessing:
		return "\033[32m"
	case StatusHumanReview:
		return "\033[31m"
	case StatusCompleted:
		return "\033[36m"
	default:
		return "\033[0m"
	}
}
