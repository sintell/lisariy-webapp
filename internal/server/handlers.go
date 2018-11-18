package server

import "github.com/labstack/echo"

func registerHandlers(e *echo.Echo) []*echo.Route {
	return []*echo.Route{
		e.Add("POST", "/login", loginHandler),
		e.Add("GET", "/login", loginHandler),
	}
}

func loginHandler(c echo.Context) error {
	return nil
}
