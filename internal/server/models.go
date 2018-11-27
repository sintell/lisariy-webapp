package server

import (
	"time"

	"github.com/satori/go.uuid"
)

type ModelDefaults struct {
	ID        uint       `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt time.Time  `json:"createdAt,omitempty"`
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

type ImageSource struct {
	SrcX1 string `json:"x1,omitempty"`
	SrcX2 string `json:"x2,omitempty"`
}

type Picture struct {
	ModelDefaults
	Title        string      `json:"title,omitempty"`
	Description  string      `json:"description,omitempty"`
	Key          uuid.UUID   `json:"-"`
	Ext          string      `json:"-"`
	OriginalSrc  string      `json:"-"`
	ThumbnailSrc ImageSource `json:"tn,omitempty"`
	ProcessedSrc ImageSource `json:"pc,omitempty"`
	Processed    bool        `json:"-"`
	Hidden       bool        `json:"isHidden,omitempty"`
}

func (p *Picture) FullName() string {
	return p.Key.String() + "." + p.Ext
}

func (p *Picture) Create() error {
	return store.db.Create(p).Error
}

func (p *Picture) GetByID(id uint) error {
	return store.db.First(p, id).Error
}

func (p *Picture) HideByID(id uint) error {
	return store.db.First(p, id).Update("hidden", true).Error
}

func (p *Picture) ShowByID(id uint) error {
	return store.db.First(p, id).Update("hidden", false).Error
}

func (p *Picture) Save() error {
	return store.db.Save(p).Error
}

func (p *Picture) Update() error {
	return store.db.Model(p).Select("title", "description").Updates(p).Error
}

func (p *Picture) DeleteByID(id uint) error {
	return store.db.Delete(p, id).Error
}

type PicturesList []*Picture

func (pl *PicturesList) GetProcessed() error {
	return store.db.Order("created_at", true).Find(pl, "processed = ? AND hidden = ?", true, false).Error
}

func (pl *PicturesList) GetAll() error {
	return store.db.Order("created_at", true).Find(pl).Error
}
