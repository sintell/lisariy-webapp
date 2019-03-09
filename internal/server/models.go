package server

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"

	"golang.org/x/crypto/bcrypt"

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
	Key          uuid.UUID `json:"key,omitempty" sql:",type:uuid"`
	Login        string    `json:"login,omitempty" sql:",unique"`
	IsAnonymous  bool      `json:"isAnonymous,omitempty"`
	PasswordHash string    `json:"-"`
}

func (u *User) String() string {
	return fmt.Sprintf("<User id='%d' login='%s' key='%s' is_anonymous='%v' password_hash='FILTERED(%d)'>",
		u.Id, u.Login, u.Key, u.IsAnonymous, len(u.PasswordHash))
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

type UserWithPassword struct {
	User
	Password string `json:"password"`
}

func (uwp *UserWithPassword) String() string {
	return fmt.Sprintf("<UserWithPassword %s password='FILTERED(%d)'>", uwp.User.String(), len(uwp.Password))
}

func (uwp *UserWithPassword) Register() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(uwp.Password), bcrypt.MinCost)
	if err != nil {
		return err
	}
	uwp.User.PasswordHash = string(hash)
	uwp.User.IsAnonymous = false

	uwp.User.Save()

	return nil
}

func (uwp *UserWithPassword) Authenticate() error {
	err := store.db.Model(&uwp.User).
		Where("login = ?login").
		First()

	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(uwp.User.PasswordHash), []byte(uwp.Password))
	if err != nil {
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

func (p *Picture) String() string {
	return fmt.Sprintf("<Picture id='%d' title='%s' original_name='%s' hidden='%v'>", p.Id, p.Title, p.FullName(), p.Hidden)
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
	return store.db.Model(p).Column("picture.*", "Tags").WherePK().Relation("Tags").First()
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
		_, err := tx.Model(p).
			Column("title", "description").
			WherePK().
			Returning("*").
			Update()
		if err != nil {
			return err
		}

		pictureToTags := []*PictureToTag{}

		_, err = tx.Model(&pictureToTags).
			Where("picture_id = ?", p.Id).
			Delete()
		if err != nil {
			return err
		}

		if p.Tags == nil || len(p.Tags) == 0 {
			return nil
		}

		for _, t := range p.Tags {
			err := t.GetOrCreate(tx)
			if err != nil {
				return err
			}
		}

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
	return store.db.RunInTransaction(func(tx *pg.Tx) error {
		p.Id = id
		pictureToTags := []*PictureToTag{}

		_, err := tx.Model(&pictureToTags).
			Where("picture_id = ?", p.Id).
			Delete()
		if err != nil {
			return err
		}

		_, err = store.db.Model(p).Delete()
		return err
	})
}

type PicturesList []*Picture

// GetAll populates pictures list with data
//	- `includeHidden` - also loads pictures that are hidden from unauthorized users
//	- `categories` - list of category ids to filter by, if nil or empty it will be ignored
func (pl *PicturesList) GetAll(includeHidden bool, categories []int) error {
	q := store.db.Model(pl).
		Column("picture.*", "Tags")

	if !includeHidden {
		q = q.Where("hidden = ?", false)
	}

	if categories != nil && len(categories) > 0 {
		q = q.Join("JOIN picture_to_tags ptt ON picture.id = ptt.picture_id").
			JoinOn("ptt.tag_id IN (?)", pg.In(categories))
	}

	err := q.
		Order("created_at ASC").
		Relation("Tags").
		Select()

	if err != nil {
		return err
	}

	for i, pic := range ([]*Picture)(*pl) {
		if pic.Tags == nil {
			([]*Picture)(*pl)[i].Tags = TagsList{}
		}
	}
	return nil
}

type Tag struct {
	ModelDefaults
	Text        string       `json:"text,omitempty"`
	Slug        string       `json:"slug,omitempty"`
	Description string       `json:"description,omitempty"`
	Hidden      bool         `json:"-"`
	Pictures    PicturesList `json:"pictures,omitempty" pg:"many2many:picture_to_tags"`
	Usages      int          `json:"usages,omitempty" sql:"-"`
}

func (t *Tag) GetOrCreate(tx *pg.Tx) error {
	tag := store.db.Model(t)
	if tx != nil {
		tag = tx.Model(t)
	}

	t.Slug = slug.Make(t.Text)

	_, err := tag.
		Where("text = ?text").
		Returning("*").
		SelectOrInsert()
	return err
}

func (t *Tag) String() string {
	return fmt.Sprintf("<Tag id='%d' text='%s' hidden='%v'>", t.Id, t.Text, t.Hidden)
}

func (t *Tag) BeforeUpdate(db orm.DB) error {
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Tag) LoadByID(id int) error {
	t.Id = id
	return store.db.Model(t).WherePK().First()
}

func (t *Tag) LoadWithPictures(id int) error {
	t.Id = id
	return store.db.
		Model(t).
		Column("tag.*", "Pictures").
		WherePK().
		First()
}

func (t *Tag) Update() error {
	t.Slug = slug.Make(t.Text)
	_, err := store.db.Model(t).
		Column("text", "description", "slug").
		WherePK().
		Returning("*").
		Update()
	return err
}

func (t *Tag) DeleteByID(id int) error {
	t.Id = id
	return store.db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := tx.Model(t).
			WherePK().
			Delete()

		if err != nil {
			return err
		}

		_, err = tx.Model(&PictureToTag{}).
			Where("tag_id = ?", t.Id).
			Delete()

		return err
	})
}

type TagsList []*Tag

func (tl *TagsList) LoadAll() error {
	return store.db.Model(tl).Order("created_at ASC").Select()
}

func (tl *TagsList) LoadAllWithCount() error {
	return store.db.
		Model(tl).
		Column("id", "updated_at", "text", "description").
		ColumnExpr("count(ptt.picture_id) AS usages").
		Join("JOIN picture_to_tags ptt ON tag.id = ptt.tag_id").
		Group("id", "updated_at", "text", "description").
		Order("usages DESC", "updated_at ASC").
		Select()
}

func (tl *TagsList) LoadWithMatchingText(text string) error {
	return store.db.Model(tl).Order("created_at ASC").Where("text LIKE ?", "%"+text+"%").Select()
}

type PictureToTag struct {
	PictureId int `sql:",pk"`
	TagId     int `sql:",pk"`
}
