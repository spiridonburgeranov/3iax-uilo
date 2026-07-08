package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/awg"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type AwgController struct {
	awgInboundService service.AwgInboundService
	inboundService    service.InboundService
}

func NewAwgController(g *gin.RouterGroup) *AwgController {
	a := &AwgController{}
	a.initRouter(g)
	return a
}

func (a *AwgController) initRouter(g *gin.RouterGroup) {
	g.GET("/provision/new", a.provisionNew)
	g.GET("/discovered", a.listDiscovered)
	g.GET("/discovered/:name/template", a.discoveredTemplate)
	g.POST("/scan/import", a.importDiscovered)
	g.GET("/inbounds", a.listInbounds)
	g.POST("/restore", a.restore)
	g.POST("/toggle", a.toggleAll)
	g.POST("/server/toggle", a.toggleAll)
	g.GET("/server/status", a.serverStatus)
	g.GET("/client/:inboundId/:email/config", a.clientConfig)
}

func (a *AwgController) provisionNew(c *gin.Context) {
	result, err := a.awgInboundService.ProvisionNew()
	jsonObj(c, result, err)
}

func (a *AwgController) listDiscovered(c *gin.Context) {
	items, err := a.awgInboundService.ListDiscovered()
	jsonObj(c, items, err)
}

func (a *AwgController) discoveredTemplate(c *gin.Context) {
	template, err := a.awgInboundService.TemplateFromInterface(c.Param("name"))
	jsonObj(c, template, err)
}

func (a *AwgController) importDiscovered(c *gin.Context) {
	var body struct {
		Force bool     `json:"force"`
		Names []string `json:"names"`
	}
	_ = c.ShouldBindJSON(&body)
	result, err := a.awgInboundService.ImportDiscovered(body.Force, body.Names)
	jsonObj(c, result, err)
}

func (a *AwgController) listInbounds(c *gin.Context) {
	a.awgInboundService.UpdateTrafficStats()
	items, err := a.awgInboundService.ListRuntime()
	jsonObj(c, items, err)
}

func (a *AwgController) restore(c *gin.Context) {
	a.awgInboundService.RestoreAll()
	jsonMsg(c, "AWG interfaces restored", nil)
}

func (a *AwgController) toggleAll(c *gin.Context) {
	var body struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG toggled", a.awgInboundService.ToggleAll(body.Enable))
}

func (a *AwgController) serverStatus(c *gin.Context) {
	runtimes, err := a.awgInboundService.ListRuntime()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	running := false
	peerCount := 0
	onlineCount := 0
	for _, item := range runtimes {
		if item.Running {
			running = true
		}
		peerCount += item.PeerCount
		onlineCount += item.OnlineCount
	}
	jsonObj(c, map[string]any{
		"running":      running,
		"awgInstalled": awg.IsInstalled(),
		"awgVersion":   awg.Version(),
		"peerCount":    peerCount,
		"onlineCount":  onlineCount,
		"inbounds":     runtimes,
	}, nil)
}

func (a *AwgController) clientConfig(c *gin.Context) {
	inboundID, err := strconv.Atoi(c.Param("inboundId"))
	if err != nil {
		jsonMsg(c, "invalid inbound id", err)
		return
	}
	email := c.Param("email")
	inbound, err := a.inboundService.GetInbound(inboundID)
	if err != nil {
		jsonObj(c, "", err)
		return
	}
	clients, err := a.inboundService.GetClients(inbound)
	if err != nil {
		jsonObj(c, "", err)
		return
	}
	var client *model.Client
	for i := range clients {
		if clients[i].Email == email {
			client = &clients[i]
			break
		}
	}
	if client == nil {
		jsonMsg(c, "client not found", err)
		return
	}
	endpoint := c.Query("endpoint")
	config, err := a.awgInboundService.ClientConfig(inbound, client, endpoint)
	jsonObj(c, config, err)
}
