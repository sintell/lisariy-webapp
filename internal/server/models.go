package server

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	uuid "github.com/satori/go.uuid"
)

func init() {
	orm.RegisterTable((*PictureToTag)(nil))
}

type ModelDefaults struct {
	Id        int        `sql:",pk" json:"id,omitempty"`
	CreatedAt time.Time  `sql:"default:now()" json:"createdAt,omitempty"`
	UpdatedAt time.Time  `sql:"default:now()" json:"-"`
	DeletedAt *time.Time `pg:",soft_delete" json:"-"`
}

type User struct {
	ModelDefaults
	Key         uuid.UUID `json:"key,omitempty" sql:",type:uuid"`
	Name        string    `json:"name,omitempty"`
	IsAnonymous bool      `json:"isAnonymous,omitempty"`
}

func (u *User) BeforeInsert(db orm.DB) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	return nil
}

func (u *User) BeforeUpdate(db orm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Create() error {
	return store.db.Insert(u)
}

func (u *User) PopulateByID(id int) error {
	u.Id = id
	return store.db.Model(u).WherePK().First()
}

func (u *User) Save() error {
	_, err := store.db.
		Model(u).
		WherePK().
		Update()
	return err
}

type ImageSource struct {
	SrcX1 string `json:"x1,omitempty"`
	SrcX2 string `json:"x2,omitempty"`
}

type Picture struct {
	ModelDefaults
	Title        string       `json:"title,omitempty"`
	Description  string       `json:"description,omitempty"`
	Key          uuid.UUID    `json:"-" sql:",type:uuid"`
	Ext          string       `json:"-"`
	OriginalSrc  string       `json:"-"`
	ThumbnailSrc *ImageSource `json:"tn,omitempty"`
	ProcessedSrc *ImageSource `json:"pc,omitempty"`
	Processed    bool         `json:"-"`
	Hidden       bool         `json:"isHidden,omitempty"`
	Tags         []*Tag       `json:"tags" pg:"many2many:picture_to_tags"`
}

func (p *Picture) FullName() string {
	return p.Key.String() + "." + p.Ext
}

func (p *Picture) Create() error {
	_, err := store.db.Model(p).Relation("Tags").Insert()
	return err
}

func (p *Picture) GetByID(id int) error {
	p.Id = id
	return store.db.Model(p).WherePK().First()
}

func (p *Picture) HideByID(id int) error {
	p.Id = id
	_, err := store.db.Model(p).
		Set("hidden = ?", true).
		WherePK().
		Returning("*").
		Update()

	return err
}

func (p *Picture) ShowByID(id int) error {
	p.Id = id
	_, err := store.db.Model(p).
		Set("hidden = ?", false).
		WherePK().
		Returning("*").
		Update()

	return err
}

func (p *Picture) Update() error {
	return store.db.RunInTransaction(func(tx *pg.Tx) error {
		for _, t := range p.Tags {
			_, err := tx.Model(t).
				Where("text = ?text").
				Returning("*").
				SelectOrInsert()
			if err != nil {
				return err
			}
		}

		_, err := tx.Model(p).
			Column("title", "description").
			WherePK().
			Returning("*").
			Update()
		if err != nil {
			return err
		}

		pictureToTags := []*PictureToTag{}
		for _, tag := range p.Tags {
			pictureToTags = append(pictureToTags, &PictureToTag{PictureId: p.Id, TagId: tag.Id})
		}
		_, err = tx.Model(&pictureToTags).
			OnConflict("(picture_id, tag_id) DO NOTHING").
			Insert()
		return err
	})
}

func (p *Picture) DeleteByID(id int) error {
	p.Id = id
	_, err := store.db.Model(p).Delete()
	return err
}

type PicturesList []*Picture

func (pl *PicturesList) GetAll() error {
	return store.db.Model(pl).
		Order("created_at ASC").
		Relation("Tags").
		Select()
}

type Tag struct {
	ModelDefaults
	Text        string       `json:"text,omitempty"`
	Description string       `json:"description,omitempty"`
	Hidden      bool         `json:"-"`
	Pictures    PicturesList `json:"pictures,omitempty"`
}

type TagsList []*Tag

func (tl *TagsList) GetAll() error {
	return store.db.Model(tl).Order("created_at ASC").Select()
}

type PictureToTag struct {
	PictureId int `sql:",pk"`
	TagId     int `sql:",pk"`
}
