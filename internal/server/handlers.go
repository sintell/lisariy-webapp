package server

import (
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/middleware"
	"github.com/sintell/lisariy-server/internal/pkg/config"

	"github.com/satori/go.uuid"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

const (
	imagesBasePath     = "/var/lib/lisariy-webapp/images"
	imagesOriginalSrc  = "/i/o"
	imagesThumbnailSrc = "/i/t"
	imagesProcessedSrc = "/i/p"
)

const (
	actionShow = "show"
	actionHide = "hide"
)

type Response struct {
	Response interface{} `json:"response,omitempty"`
	Error    interface{} `json:"error,omitempty"`
}

func registerHandlers(e *echo.Echo, cfg *config.Config) {
	e.Add(http.MethodGet, "/api/pictures", picturesListHandler).Name = "pictures"
	e.Add(http.MethodGet, "/api/picture/:id", pictureHandler).Name = "picture"
	e.Add(http.MethodGet, "/api/categories", categoriesListhandler).Name = "categories"
	e.Add(http.MethodGet, "/api/category/:id", categoryHandler).Name = "category"
	e.Add(http.MethodGet, "/api/category", categorySearchHandler).Name = "categorySearch"
	e.Add(http.MethodPost, "/api/login", loginHandler).Name = "login"
	e.Add(http.MethodPost, "/api/admin/register", registerAdminHandler, middleware.BasicAuth(
		func(l, p string, c echo.Context) (bool, error) {
			if l == cfg.WebApp.AdminLogin && p == cfg.WebApp.AdminPassword {
				return true, nil
			}
			return false, nil
		}))

	protected := e.Group("", checkAuth)
	protected.Add(http.MethodGet, "/api/me", meHandler)
	protected.Add(http.MethodPost, "/api/logout", logoutHandler).Name = "logout"
	protected.Add(http.MethodPost, "/api/pictures", newPictureHandler).Name = "newPicture"
	protected.Add(http.MethodPut, "/api/picture/:id/:action", pictureVisibilityHandler)
	protected.Add(http.MethodPut, "/api/picture/:id", pictureUpdateHandler)
	protected.Add(http.MethodDelete, "/api/picture/:id", pictureDeleteHandler)
}

func loginHandler(c echo.Context) error {
	u, err := userFromSession(getSession(c))
	if err != nil {
		return c.JSON(http.StatusBadRequest, struct{ Error string }{
			"no user in session: " + err.Error(),
		})
	}

	uwp := &UserWithPassword{User: *u}
	err = c.Bind(uwp)
	if err != nil {
		c.Logger().Warn("Error parsing creds:", err)
		return c.JSON(http.StatusBadRequest, Response{Error: "Can't parse credentials"})
	}

	c.Logger().Debugf("got user data: %s", uwp)

	err = uwp.Authenticate()
	if err != nil {
		c.Logger().Warn("Error authenticating:", err)
		return c.JSON(http.StatusForbidden, Response{Error: "Bad credentials"})
	}

	u.IsAnonymous = false
	u.Save()

	return c.JSON(http.StatusOK, Response{Response: u})
}

func logoutHandler(c echo.Context) error {
	sess := getSession(c)
	u, err := userFromSession(sess)
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: "No user in session"})
	}

	u.IsAnonymous = true
	u.Save()
	sess.Save(c.Request(), c.Response())
	return nil
}

func meHandler(c echo.Context) error {
	u, err := userFromSession(getSession(c))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Error: "no user in session",
		})
	}

	return c.JSON(http.StatusOK, Response{Response: u})
}

func registerAdminHandler(c echo.Context) error {
	u, _ := userFromSession(getSession(c))
	uwp := &UserWithPassword{User: *u}
	err := c.Bind(uwp)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, Response{Error: "Bad credentials"})
	}

	err = uwp.Register()
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Something went wrong with the server")
	}

	return c.JSON(http.StatusOK, uwp.User)
}

func picturesListHandler(c echo.Context) error {
	pl := &PicturesList{}
	if err := pl.GetAll(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}
	return c.JSON(http.StatusOK, Response{Response: pl})
}

func pictureHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	p := &Picture{}
	if err := p.GetByID(id); err != nil {
		if err == ErrNotFound {
			return c.JSON(http.StatusNotFound, Response{Error: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}
	return c.JSON(http.StatusOK, Response{Response: p})
}

func newPictureHandler(c echo.Context) error {
	fileErrors := make(map[string]string)
	createdFiles := make(map[int]*Picture)

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	c.Logger().Debugf("got files to store: %v", form.File)
	files := form.File["files"]
	t := time.Now()
	datePath := path.Join(
		strconv.Itoa(t.Year()),
		strconv.Itoa(int(t.Month())),
		strconv.Itoa(t.Day()))
	err = os.MkdirAll(path.Join(imagesBasePath, imagesOriginalSrc, datePath), 0755)
	err = os.MkdirAll(path.Join(imagesBasePath, imagesThumbnailSrc, datePath), 0755)
	err = os.MkdirAll(path.Join(imagesBasePath, imagesProcessedSrc, datePath), 0755)
	if err != nil {
		c.Logger().Error("error creating dirs:", err.Error())
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	syncPoint := make(chan interface{}, 100)

	for _, file := range files {
		key := uuid.NewV4()
		tnSrc := path.Join(imagesThumbnailSrc, datePath, uuid.NewV4().String()+path.Ext(file.Filename))
		tnX2Src := path.Join(imagesThumbnailSrc, datePath, uuid.NewV4().String()+"@2x"+path.Ext(file.Filename))
		pcSrc := path.Join(imagesProcessedSrc, datePath, uuid.NewV4().String()+path.Ext(file.Filename))
		pcX2Src := path.Join(imagesProcessedSrc, datePath, uuid.NewV4().String()+"@2x"+path.Ext(file.Filename))
		pic := &Picture{
			Hidden:       true,
			Key:          key,
			Ext:          path.Ext(file.Filename),
			OriginalSrc:  path.Join(imagesOriginalSrc, datePath, key.String()+path.Ext(file.Filename)),
			ThumbnailSrc: &ImageSource{SrcX1: tnSrc, SrcX2: tnX2Src},
			ProcessedSrc: &ImageSource{SrcX1: pcSrc, SrcX2: pcX2Src},
			Tags:         []*Tag{&Tag{Text: "Deafult Category"}},
		}

		c.Logger().Debugf("file header is: %+v", file.Header)

		if !strings.Contains(file.Header.Get("Content-Type"), "image") {
			c.Logger().Errorf("file is not an image: %s but an %s", file.Filename, file.Header.Get("Content-Type"))
			fileErrors[file.Filename] = "file is not an image"
			continue
		}

		src, err := file.Open()
		if err != nil {
			c.Logger().Errorf("can't read file %s from request, reason: %s", file.Filename, err)
			fileErrors[file.Filename] = err.Error()
			continue
		}
		defer src.Close()

		pth := path.Join(imagesBasePath, pic.OriginalSrc)
		dst, err := os.Create(pth)
		if err != nil {
			c.Logger().Errorf("can't create file %s at path %s, reason: %s", file.Filename, pth, err)
			fileErrors[file.Filename] = err.Error()
			continue
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			c.Logger().Errorf("can't copy file %s at path %s, reason: %s", file.Filename, pth, err)
			fileErrors[file.Filename] = err.Error()
			continue
		}
		syncPoint <- pp.PutOriginal(pic)

		err = pic.Create()
		if err != nil {
			c.Logger().Errorf("can't create file %s metadata to DB: %s", file.Filename, err)
			fileErrors[file.Filename] = err.Error()
			continue
		}
		createdFiles[pic.Id] = pic
	}

	if len(fileErrors) > 0 {
		c.Logger().Error(fileErrors)
		return c.JSON(http.StatusBadRequest, Response{Error: &fileErrors})
	}

	syncs := len(files)
	i := 0
	for {
		if i >= syncs {
			break
		}
		select {
		case <-syncPoint:
			i++
			c.Logger().Debugf("syncPoint increment %d of %d", i, syncs)
		default:
		}
	}

	return c.JSON(http.StatusCreated, Response{Response: &createdFiles})
}

func pictureVisibilityHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	action := c.Param("action")
	p := &Picture{}

	switch action {
	case actionShow:
		if err := p.ShowByID(id); err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
	case actionHide:
		if err := p.HideByID(id); err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
	default:
		return c.String(http.StatusNotFound, "Not Found")
	}

	return c.String(http.StatusCreated, "changed")
}

func pictureUpdateHandler(c echo.Context) error {
	p := &Picture{}
	c.Bind(p)
	c.Logger().Debugf("picture body %v", p)
	if err := p.Update(); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}

	c.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func pictureDeleteHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	p := &Picture{}
	if err := p.DeleteByID(id); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}

	c.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}
		u, err := userFromSession(sess)
		if err != nil {
			return c.JSON(http.StatusBadRequest, struct{ Error string }{
				"no user in session",
			})
		}
		if u.IsAnonymous {
			return c.JSON(http.StatusForbidden, struct{ Error string }{
				"user is unauthorised",
			})
		}
		return next(c)
	}
}

func categoriesListhandler(c echo.Context) error {
	categories := TagsList{}

	err := categories.LoadAllWithCount()
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "Service unavailable")
	}

	return c.JSON(http.StatusOK, Response{Response: categories})
}

func categoryHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	category := Tag{}
	err = category.LoadByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{Error: "Not found"})
	}

	return c.JSON(http.StatusOK, Response{Response: category})
}

func categorySearchHandler(c echo.Context) error {
	query := c.QueryParam("text")
	categories := TagsList{}
	err := categories.LoadWithMatchingText(query)
	if err != nil {
		return c.JSON(http.StatusOK, Response{Response: []*Tag{}})
	}

	return c.JSON(http.StatusOK, Response{Response: categories})
}
