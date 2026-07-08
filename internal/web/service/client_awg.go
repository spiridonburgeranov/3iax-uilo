package service

import (
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func syncAwgClientFromInboundClient(client *model.Client) error {
	if client == nil {
		return nil
	}
	awgClient := &model.AwgClient{
		UUID:       strings.TrimSpace(client.ID),
		Email:      strings.TrimSpace(client.Email),
		Name:       strings.TrimSpace(client.Email),
		Comment:    client.Comment,
		Enable:     client.Enable,
		TotalGB:    client.TotalGB,
		ExpiryTime: client.ExpiryTime,
		Reset:      client.Reset,
		LimitIp:    client.LimitIp,
		TgId:       client.TgId,
	}
	if existing, err := (&AwgService{}).GetClientByUUID(awgClient.UUID); err == nil {
		existing.Email = awgClient.Email
		existing.Name = awgClient.Name
		existing.Comment = awgClient.Comment
		existing.Enable = awgClient.Enable
		existing.TotalGB = awgClient.TotalGB
		existing.ExpiryTime = awgClient.ExpiryTime
		existing.Reset = awgClient.Reset
		existing.LimitIp = awgClient.LimitIp
		existing.TgId = awgClient.TgId
		return (&AwgService{}).UpdateClient(existing)
	}
	return (&AwgService{}).AddClient(awgClient)
}
