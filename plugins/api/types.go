package api

import (
	"github.com/labstack/echo/v4"
	glowdTypes "github.com/scitags/flowd-go/types"
)

const (
	JSON_PRETTY_INDENT string = "    "
)

type rootResponse struct {
	ApiRoutes []*echo.Route
}

type extendedContext struct {
	echo.Context
	apiRoutes   []*echo.Route
	flowChannel chan<- glowdTypes.FlowID
}
