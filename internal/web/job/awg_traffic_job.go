package job

import (
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/websocket"
)

type AwgTrafficJob struct {
	awgInboundService service.AwgInboundService
	inboundService    service.InboundService
}

func NewAwgTrafficJob() *AwgTrafficJob {
	return new(AwgTrafficJob)
}

func (j *AwgTrafficJob) Run() {
	poll := j.awgInboundService.PollTrafficStats()
	j.inboundService.RefreshLocalOnlineClients(poll.OnlineEmails, poll.ActiveInboundTags)
	if !websocket.HasClients() {
		return
	}
	onlineClients := j.inboundService.GetOnlineClients()
	if onlineClients == nil {
		onlineClients = []string{}
	}
	payload := map[string]any{
		"onlineClients":  onlineClients,
		"onlineByGuid":   j.inboundService.GetOnlineClientsByGuid(),
		"activeInbounds": j.inboundService.GetActiveInboundsByGuid(),
	}
	if len(poll.ClientDeltas) > 0 {
		payload["clientTraffics"] = poll.ClientDeltas
		logger.Debugf("awg traffic poll: %d client delta(s)", len(poll.ClientDeltas))
	}
	websocket.BroadcastTraffic(payload)
}
