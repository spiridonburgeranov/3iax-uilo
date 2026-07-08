package job

import "github.com/mhsanaei/3x-ui/v3/internal/web/service"

type AwgTrafficJob struct {
	awgInboundService service.AwgInboundService
}

func NewAwgTrafficJob() *AwgTrafficJob {
	return new(AwgTrafficJob)
}

func (j *AwgTrafficJob) Run() {
	j.awgInboundService.UpdateTrafficStats()
}
