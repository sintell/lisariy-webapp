package server

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

type Response struct {
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
}

func registerHandlers(e *echo.Echo) {
	e.Add("POST", "/login", loginHandler).Name = "login"
}

func registerProtectedHandlers(e *echo.Echo) {
	protected := e.Group("", checkAuth)
	protected.Add("POST", "/logout", logoutHandler).Name = "logout"
	protected.Add("GET", "/me", meHandler)
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

	return nil
}

func logoutHandler(c echo.Context) error {
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
	u.IsAnonymous = true
	sess.Save(c.Request(), c.Response())
	return nil
}

func meHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	u, err := userFromSession(sess)
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Error: "no user in session",
		})
	}
	return c.JSON(http.StatusOK, Response{Response: u})
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
