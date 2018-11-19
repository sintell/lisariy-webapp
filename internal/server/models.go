package server

import (
	"time"

	"github.com/satori/go.uuid"
)

type ModelDefaults struct {
	ID        uint       `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

type User struct {
	ModelDefaults
	Key         uuid.UUID `json:"key,omitempty"`
	Name        string    `json:"name,omitempty"`
	IsAnonymous bool      `json:"isAnonymous,omitempty"`
}

func (u *User) Create() error {
	if err := store.db.Create(u).Error; err != nil {
		return err
	}
	return nil
}

func (u *User) PopulateByID(id uint) error {
	if err := store.db.First(u, id).Error; err != nil {
		return err
	}
	return nil
}

func (u *User) Save() error {
	if err := store.db.Save(u).Error; err != nil {
		return err
	}

	return nil
}
