package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type AwgController struct {
	awgService service.AwgService
}

func NewAwgController(g *gin.RouterGroup) *AwgController {
	a := &AwgController{}
	a.initRouter(g)
	return a
}

func (a *AwgController) initRouter(g *gin.RouterGroup) {
	g.GET("/server", a.getServer)
	g.POST("/server", a.saveServer)
	g.POST("/server/toggle", a.toggleServer)
	g.GET("/server/status", a.getServerStatus)
	g.GET("/clients", a.getClients)
	g.POST("/client/add", a.addClient)
	g.POST("/client/update/:id", a.updateClient)
	g.POST("/client/updateByUuid/:uuid", a.updateClientByUUID)
	g.POST("/client/del/:id", a.deleteClient)
	g.POST("/client/delByUuid/:uuid", a.deleteClientByUUID)
	g.POST("/client/toggle/:id", a.toggleClient)
	g.POST("/client/toggleByUuid/:uuid", a.toggleClientByUUID)
	g.POST("/client/reissue/:id", a.reissueClient)
	g.GET("/client/:id/config", a.getClientConfig)
	g.GET("/client/uuid/:uuid/config", a.getClientConfigByUUID)
}

func (a *AwgController) getServer(c *gin.Context) {
	server, err := a.awgService.GetServer()
	jsonObj(c, server, err)
}

func (a *AwgController) saveServer(c *gin.Context) {
	var server model.AwgServer
	if err := c.ShouldBindJSON(&server); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG server settings saved", a.awgService.SaveServer(&server))
}

func (a *AwgController) toggleServer(c *gin.Context) {
	var body struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG server toggled", a.awgService.ToggleServer(body.Enable))
}

func (a *AwgController) getServerStatus(c *gin.Context) {
	jsonObj(c, a.awgService.GetServerStatus(), nil)
}

func (a *AwgController) getClients(c *gin.Context) {
	a.awgService.UpdateTrafficStats()
	clients, err := a.awgService.GetClients()
	jsonObj(c, clients, err)
}

func (a *AwgController) addClient(c *gin.Context) {
	var client model.AwgClient
	if err := c.ShouldBindJSON(&client); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	if err := a.awgService.AddClient(&client); err != nil {
		jsonMsg(c, "add AWG client", err)
		return
	}
	jsonObj(c, client, nil)
}

func (a *AwgController) updateClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid id", err)
		return
	}
	var client model.AwgClient
	if err := c.ShouldBindJSON(&client); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	client.Id = id
	jsonMsg(c, "AWG client updated", a.awgService.UpdateClient(&client))
}

func (a *AwgController) updateClientByUUID(c *gin.Context) {
	clientUUID := c.Param("uuid")
	if clientUUID == "" {
		jsonMsg(c, "invalid uuid", errors.New("missing uuid"))
		return
	}
	var client model.AwgClient
	if err := c.ShouldBindJSON(&client); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG client updated", a.awgService.UpdateClientByUUID(clientUUID, &client))
}

func (a *AwgController) deleteClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid id", err)
		return
	}
	jsonMsg(c, "AWG client deleted", a.awgService.DeleteClient(id))
}

func (a *AwgController) deleteClientByUUID(c *gin.Context) {
	clientUUID := c.Param("uuid")
	if clientUUID == "" {
		jsonMsg(c, "invalid uuid", errors.New("missing uuid"))
		return
	}
	jsonMsg(c, "AWG client deleted", a.awgService.DeleteClientByUUID(clientUUID))
}

func (a *AwgController) toggleClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid id", err)
		return
	}
	var body struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG client toggled", a.awgService.ToggleClient(id, body.Enable))
}

func (a *AwgController) toggleClientByUUID(c *gin.Context) {
	clientUUID := c.Param("uuid")
	if clientUUID == "" {
		jsonMsg(c, "invalid uuid", errors.New("missing uuid"))
		return
	}
	var body struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	jsonMsg(c, "AWG client toggled", a.awgService.ToggleClientByUUID(clientUUID, body.Enable))
}

func (a *AwgController) reissueClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid id", err)
		return
	}
	client, err := a.awgService.ReissueClient(id)
	jsonObj(c, client, err)
}

func (a *AwgController) getClientConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid id", err)
		return
	}
	config, err := a.awgService.GetClientConfig(id)
	jsonObj(c, config, err)
}

func (a *AwgController) getClientConfigByUUID(c *gin.Context) {
	clientUUID := c.Param("uuid")
	if clientUUID == "" {
		jsonMsg(c, "invalid uuid", errors.New("missing uuid"))
		return
	}
	config, err := a.awgService.GetClientConfigByUUID(clientUUID)
	jsonObj(c, config, err)
}
