package database

import (
	"time"
)

type Users struct {
	Id           int
	Username     string `gorm:"unique"`
	Email        string `gorm:"unique"`
	PasswordHash string
	CreatedAt    time.Time
	LastSeen     time.Time
	AvatarUrl    string
}

type Notifications struct {
	Id        int
	UserID    int
	Message   string
	CreatedAt time.Time
	IsRead    bool
}

type Messages struct {
	Id        int
	ChatID    int
	SenderID  int
	Context   string
	FileURL   string
	CreatedAt time.Time
}

type Chats struct {
	Id        int
	Name      string
	IsGroup   bool
	CreatedAt time.Time
}

type ChatMembers struct {
	Id        int
	Name      string
	IsGroup   bool
	CreatedAt time.Time
}

type Calls struct {
	Id         int
	CallerID   int
	ReceiverID int
	ChatID     int
	StartedAt  time.Time
	EndedAt    time.Time
	Status     string
	CallType   string
}
