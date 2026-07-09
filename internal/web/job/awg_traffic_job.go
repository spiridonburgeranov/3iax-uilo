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
	if len(poll.InboundTraffics) > 0 {
		if _, _, err := j.inboundService.AddTraffic(poll.InboundTraffics, nil); err != nil {
			logger.Warning("awg traffic job: add inbound traffic failed:", err)
		}
	}
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
	if len(poll.InboundTraffics) > 0 {
		payload["traffics"] = poll.InboundTraffics
	}
	if len(poll.ClientDeltas) > 0 {
		payload["clientTraffics"] = poll.ClientDeltas
		logger.Debugf("awg traffic poll: %d client delta(s)", len(poll.ClientDeltas))
	}
	websocket.BroadcastTraffic(payload)

	if summary, err := j.inboundService.GetInboundsTrafficSummary(); err != nil {
		logger.Warning("awg traffic job: inbound summary for websocket failed:", err)
	} else if len(summary) > 0 {
		websocket.BroadcastClientStats(map[string]any{
			"snapshot": false,
			"inbounds": summary,
		})
	}
}
