package api

import (
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	glowdTypes "github.com/scitags/flowd-go/types"
)

var dummyFlowID = glowdTypes.FlowID{
	// State:    glowd.START,
	Protocol: glowdTypes.TCP,
	Src: glowdTypes.IPPort{
		IP:   net.ParseIP("::1"),
		Port: 2345,
	},
	Dst: glowdTypes.IPPort{
		IP:   net.ParseIP("::1"),
		Port: 5777,
	},
	Activity:   0xFFFF,
	Experiment: 0xFFFF,
	// StartTs:    time.Now(),
}

func handleRoot(c echo.Context) error {
	cc := c.(*extendedContext)
	return c.JSONPretty(http.StatusOK, &rootResponse{
		ApiRoutes: cc.apiRoutes,
	}, JSON_PRETTY_INDENT)
}

func handleDummyStartFlow(c echo.Context) error {
	cc := c.(*extendedContext)

	tmp := dummyFlowID
	tmp.State = glowdTypes.START
	tmp.StartTs = time.Now()

	cc.flowChannel <- tmp
	return c.JSONPretty(http.StatusOK, &tmp, JSON_PRETTY_INDENT)
}

func handleDummyEndFlow(c echo.Context) error {
	cc := c.(*extendedContext)

	tmp := dummyFlowID
	tmp.State = glowdTypes.END
	tmp.EndTs = time.Now()

	cc.flowChannel <- tmp
	return c.JSONPretty(http.StatusOK, &tmp, JSON_PRETTY_INDENT)
}
