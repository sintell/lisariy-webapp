package server

import (
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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

func registerHandlers(e *echo.Echo) {
	e.Add(http.MethodGet, "/api/pictures", picturesListHandler).Name = "pictures"
	e.Add(http.MethodGet, "/api/picture/:id", pictureHandler).Name = "picture"
	e.Add(http.MethodPost, "/api/login", loginHandler).Name = "login"
}

func registerProtectedHandlers(e *echo.Echo) {
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
			"no user in session",
		})
	}
	u.IsAnonymous = false
	u.Save()

	return c.JSON(http.StatusOK, Response{Response: u})
}

func logoutHandler(c echo.Context) error {
	sess := getSession(c)
	u, err := userFromSession(sess)
	if err != nil {
		return c.JSON(http.StatusBadRequest, struct{ Error string }{
			"no user in session",
		})
	}
	u.IsAnonymous = true
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
	if err := p.GetByID(uint(id)); err != nil {
		if err == ErrNotFound {
			return c.JSON(http.StatusNotFound, Response{Error: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}
	return c.JSON(http.StatusOK, Response{Response: p})
}

func newPictureHandler(c echo.Context) error {
	fileErrors := make(map[string]string)
	createdFiles := make(map[uint]*Picture)

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

	for _, file := range files {
		key := uuid.NewV4()
		pic := &Picture{
			Hidden:       true,
			Key:          key,
			Ext:          path.Ext(file.Filename),
			OriginalSrc:  path.Join(imagesOriginalSrc, datePath, key.String()+path.Ext(file.Filename)),
			ThumbnailSrc: path.Join(imagesThumbnailSrc, datePath, uuid.NewV4().String()+path.Ext(file.Filename)),
			ProcessedSrc: path.Join(imagesProcessedSrc, datePath, uuid.NewV4().String()+path.Ext(file.Filename)),
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
		pp.PutOriginal(pic)

		err = pic.Create()
		if err != nil {
			c.Logger().Errorf("can't create file %s metadata to DB: %s", file.Filename, err)
			fileErrors[file.Filename] = err.Error()
			continue
		}
		createdFiles[pic.ID] = pic
	}
	if len(fileErrors) > 0 {
		return c.JSON(http.StatusBadRequest, Response{Error: &fileErrors})
	}

	return c.JSON(http.StatusCreated, Response{Response: &createdFiles})
}

func pictureVisibilityHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	action := c.Param("action")
	p := &Picture{}

	switch action {
	case actionShow:
		if err := p.ShowByID(uint(id)); err != nil {
			return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
	case actionHide:
		if err := p.HideByID(uint(id)); err != nil {
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
		return c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
	}

	c.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func pictureDeleteHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	p := &Picture{}
	if err := p.DeleteByID(uint(id)); err != nil {
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
