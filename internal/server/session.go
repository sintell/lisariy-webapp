package server

import (
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	uuid "github.com/satori/go.uuid"
)

type sessionKeys string

const (
	KeyUser       = "user"
	KeyAuthedUser = "authed-user"
)

func getSession(c echo.Context) *sessions.Session {
	sess, err := session.Get("session", c)
	if err != nil {
		c.Logger().Errorf("no session in context: %s", err)
		c.String(http.StatusInternalServerError, err.Error())
		return nil
	}

	return sess
}

func setupSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Logger().Debug("setting up session")
		sess := getSession(c)
		c.Logger().Debug("session values: %v", sess.Values)

		if _, exists := sess.Values[KeyUser]; exists {
			if u, err := userFromSession(c); err == nil {
				if !u.IsAnonymous {
					c.Set(KeyAuthedUser, &u)
				}
				return next(c)
			}
			c.Logger().Warnf("stale session: %v, will recreate", sess)
		}

		u := &User{IsAnonymous: true, Key: uuid.NewV4()}
		if err := u.Create(); err != nil {
			c.Logger().Error("can't create new user: ", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		sess.Values[KeyUser] = u.Id
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			c.Logger().Error("can't save session:", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		c.Logger().Debugf("session set for %v", u)
		return next(c)
	}
}

func userFromSession(c echo.Context) (*User, error) {
	if u, ok := c.Get(KeyAuthedUser).(User); ok {
		return &u, nil
	}

	s := getSession(c)
	if uid, exists := s.Values[KeyUser]; exists {
		u := &User{}
		if err := u.PopulateByID(uid.(int)); err != nil {
			return nil, err
		}

		return u, nil
	}

	return nil, errors.New("no user present")
}

func setUserToSession(c echo.Context, u *User) error {
	sess := getSession(c)
	sess.Values[KeyUser] = u.Id
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Error("can't save session:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Logger().Debugf("session set for %v", u)
	return nil
}
