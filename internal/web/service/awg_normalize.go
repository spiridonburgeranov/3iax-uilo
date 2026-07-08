package service

import (
	"github.com/mhsanaei/3x-ui/v3/internal/awg"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func (s *InboundService) normalizeAmneziawgInbound(inbound *model.Inbound) error {
	if inbound == nil || inbound.Protocol != model.AmneziaWG || inbound.NodeID != nil {
		return nil
	}
	snapshot := (&AwgInboundService{}).buildResourceSnapshot()
	normalized, port, err := awg.NormalizeInboundSettings(inbound.Settings, inbound.Port, snapshot)
	if err != nil {
		return err
	}
	inbound.Settings = normalized
	if inbound.Port <= 0 && port > 0 {
		inbound.Port = port
	}
	iface, err := awg.ParseInboundSettings(normalized)
	if err == nil && inbound.Tag == "" && iface.AwgInterface != "" {
		inbound.Tag = awg.FormatInterfaceTag(iface.AwgInterface)
	}
	return nil
}
