package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/awg"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"gorm.io/gorm"
)

type AwgService struct{}

type AwgStatus struct {
	Running      bool   `json:"running"`
	AwgInstalled bool   `json:"awgInstalled"`
	AwgVersion   string `json:"awgVersion"`
}

func (s *AwgService) GetServer() (*model.AwgServer, error) {
	db := database.GetDB()
	var server model.AwgServer
	if err := db.FirstOrCreate(&server).Error; err != nil {
		return nil, err
	}
	changed := false
	if strings.TrimSpace(server.InterfaceName) == "" {
		server.InterfaceName = "awg0"
		changed = true
	}
	if server.ListenPort <= 0 {
		server.ListenPort = 51820
		changed = true
	}
	if server.MTU <= 0 {
		server.MTU = 1420
		changed = true
	}
	if strings.TrimSpace(server.IPv4Address) == "" {
		server.IPv4Address = "10.66.66.1/24"
		changed = true
	}
	if strings.TrimSpace(server.IPv4Pool) == "" {
		server.IPv4Pool = "10.66.66.0/24"
		changed = true
	}
	if strings.TrimSpace(server.DNS) == "" {
		server.DNS = "1.1.1.1,2606:4700:4700::1111"
		changed = true
	}
	if strings.TrimSpace(server.PrivateKey) == "" || strings.TrimSpace(server.PublicKey) == "" {
		priv, pub, err := awg.GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("generate awg server keys: %w", err)
		}
		server.PrivateKey = priv
		server.PublicKey = pub
		changed = true
	}
	if server.Jc <= 0 {
		server.Jc = 4
		changed = true
	}
	if server.Jmin <= 0 {
		server.Jmin = 64
		changed = true
	}
	if server.Jmax <= 0 {
		server.Jmax = 256
		changed = true
	}
	if server.S1 <= 0 {
		server.S1 = 15
		changed = true
	}
	if server.S2 <= 0 {
		server.S2 = 25
		changed = true
	}
	if server.S3 <= 0 {
		server.S3 = 35
		changed = true
	}
	if server.S4 <= 0 {
		server.S4 = 15
		changed = true
	}
	if server.H1 <= 0 {
		server.H1 = 5
		changed = true
	}
	if server.H2 <= 0 {
		server.H2 = 10
		changed = true
	}
	if server.H3 <= 0 {
		server.H3 = 15
		changed = true
	}
	if server.H4 <= 0 {
		server.H4 = 20
		changed = true
	}
	if changed {
		if err := db.Save(&server).Error; err != nil {
			return nil, err
		}
	}
	return &server, nil
}

func (s *AwgService) SaveServer(server *model.AwgServer) error {
	if strings.TrimSpace(server.InterfaceName) == "" {
		server.InterfaceName = "awg0"
	}
	if server.ListenPort <= 0 {
		server.ListenPort = 51820
	}
	if strings.TrimSpace(server.PrivateKey) == "" || strings.TrimSpace(server.PublicKey) == "" {
		priv, pub, err := awg.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("generate awg server keys: %w", err)
		}
		server.PrivateKey = priv
		server.PublicKey = pub
	}
	server.UpdatedAt = time.Now().UnixMilli()
	if err := database.GetDB().Save(server).Error; err != nil {
		return err
	}
	if server.Enable {
		return s.applyServerConfig(server)
	}
	return nil
}

func (s *AwgService) ToggleServer(enable bool) error {
	server, err := s.GetServer()
	if err != nil {
		return err
	}
	if err := database.GetDB().Model(server).Update("enable", enable).Error; err != nil {
		return err
	}
	server.Enable = enable
	if enable {
		return s.applyServerConfig(server)
	}
	return awg.InterfaceDown(server.InterfaceName)
}

func (s *AwgService) GetServerStatus() *AwgStatus {
	server, _ := s.GetServer()
	iface := "awg0"
	if server != nil && strings.TrimSpace(server.InterfaceName) != "" {
		iface = server.InterfaceName
	}
	return &AwgStatus{
		Running:      awg.IsInterfaceUp(iface),
		AwgInstalled: awg.IsInstalled(),
		AwgVersion:   awg.Version(),
	}
}

func (s *AwgService) GetClients() ([]model.AwgClient, error) {
	var clients []model.AwgClient
	err := database.GetDB().Order("id asc").Find(&clients).Error
	return clients, err
}

func (s *AwgService) GetClient(id int) (*model.AwgClient, error) {
	var client model.AwgClient
	if err := database.GetDB().First(&client, id).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *AwgService) GetClientByUUID(clientUUID string) (*model.AwgClient, error) {
	var client model.AwgClient
	if err := database.GetDB().Where("uuid = ?", clientUUID).First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *AwgService) AddClient(client *model.AwgClient) error {
	server, err := s.GetServer()
	if err != nil {
		return err
	}
	if strings.TrimSpace(client.UUID) == "" {
		client.UUID = uuid.NewString()
	}
	if _, err := uuid.Parse(client.UUID); err != nil {
		return fmt.Errorf("invalid awg client uuid: %w", err)
	}
	if strings.TrimSpace(client.Email) == "" {
		client.Email = client.UUID
	}
	if strings.TrimSpace(client.Name) == "" {
		client.Name = client.Email
	}
	var count int64
	if err := database.GetDB().Model(&model.AwgClient{}).
		Where("uuid = ? OR email = ?", client.UUID, client.Email).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("awg client already exists")
	}
	priv, pub, err := awg.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate awg client keys: %w", err)
	}
	psk, err := awg.GeneratePresharedKey()
	if err != nil {
		return fmt.Errorf("generate awg psk: %w", err)
	}
	client.PrivateKey = priv
	client.PublicKey = pub
	client.PresharedKey = psk
	existing, err := s.GetClients()
	if err != nil {
		return err
	}
	usedIPv4 := make([]string, 0, len(existing))
	usedIPv6 := make([]string, 0, len(existing))
	for _, existingClient := range existing {
		usedIPv4 = append(usedIPv4, existingClient.IPv4Address)
		if existingClient.IPv6Address != "" {
			usedIPv6 = append(usedIPv6, existingClient.IPv6Address)
		}
	}
	ipv4, err := awg.AllocateIPv4(server.IPv4Pool, server.IPv4Address, usedIPv4)
	if err != nil {
		return err
	}
	client.IPv4Address = ipv4
	if server.IPv6Enabled && strings.TrimSpace(server.IPv6Pool) != "" {
		ipv6, ipErr := awg.AllocateIPv6(server.IPv6Pool, server.IPv6Address, usedIPv6)
		if ipErr != nil {
			return ipErr
		}
		client.IPv6Address = ipv6
	}
	client.AllowedIPs = client.IPv4Address
	if strings.TrimSpace(client.IPv6Address) != "" {
		client.AllowedIPs += ", " + client.IPv6Address
	}
	if strings.TrimSpace(client.ClientAllowedIPs) == "" {
		client.ClientAllowedIPs = "0.0.0.0/0, ::/0"
	}
	if client.Jc <= 0 {
		client.Jc = server.Jc
	}
	if client.Jmin <= 0 {
		client.Jmin = server.Jmin
	}
	if client.Jmax <= 0 {
		client.Jmax = server.Jmax
	}
	if strings.TrimSpace(client.I1) == "" {
		client.I1 = server.I1
	}
	if strings.TrimSpace(client.I2) == "" {
		client.I2 = server.I2
	}
	if strings.TrimSpace(client.I3) == "" {
		client.I3 = server.I3
	}
	if strings.TrimSpace(client.I4) == "" {
		client.I4 = server.I4
	}
	if strings.TrimSpace(client.I5) == "" {
		client.I5 = server.I5
	}
	if client.PersistentKeepalive <= 0 {
		client.PersistentKeepalive = 25
	}
	client.ServerId = server.Id
	client.CreatedAt = time.Now().UnixMilli()
	if err := database.GetDB().Create(client).Error; err != nil {
		return err
	}
	if !server.Enable {
		_ = database.GetDB().Model(server).Update("enable", true).Error
		server.Enable = true
	}
	if err := s.applyServerConfig(server); err != nil {
		logger.Warning("apply awg config after client add failed:", err)
	}
	return nil
}

func (s *AwgService) UpdateClient(client *model.AwgClient) error {
	var old model.AwgClient
	hasOld := client.Id > 0 && database.GetDB().First(&old, client.Id).Error == nil
	if hasOld {
		if strings.TrimSpace(client.UUID) == "" {
			client.UUID = old.UUID
		}
		if client.CreatedAt <= 0 {
			client.CreatedAt = old.CreatedAt
		}
		if strings.TrimSpace(client.PrivateKey) == "" {
			client.PrivateKey = old.PrivateKey
		}
		if strings.TrimSpace(client.PublicKey) == "" {
			client.PublicKey = old.PublicKey
		}
		if strings.TrimSpace(client.PresharedKey) == "" {
			client.PresharedKey = old.PresharedKey
		}
		if client.Jc <= 0 {
			client.Jc = old.Jc
		}
		if client.Jmin <= 0 {
			client.Jmin = old.Jmin
		}
		if client.Jmax <= 0 {
			client.Jmax = old.Jmax
		}
		if strings.TrimSpace(client.I1) == "" {
			client.I1 = old.I1
		}
		if strings.TrimSpace(client.I2) == "" {
			client.I2 = old.I2
		}
		if strings.TrimSpace(client.I3) == "" {
			client.I3 = old.I3
		}
		if strings.TrimSpace(client.I4) == "" {
			client.I4 = old.I4
		}
		if strings.TrimSpace(client.I5) == "" {
			client.I5 = old.I5
		}
		if strings.TrimSpace(client.IPv4Address) == "" {
			client.IPv4Address = old.IPv4Address
		}
		if strings.TrimSpace(client.IPv6Address) == "" {
			client.IPv6Address = old.IPv6Address
		}
		if strings.TrimSpace(client.AllowedIPs) == "" {
			client.AllowedIPs = old.AllowedIPs
		}
		if strings.TrimSpace(client.ClientAllowedIPs) == "" {
			client.ClientAllowedIPs = old.ClientAllowedIPs
		}
		if client.PersistentKeepalive <= 0 {
			client.PersistentKeepalive = old.PersistentKeepalive
		}
		if client.ServerId <= 0 {
			client.ServerId = old.ServerId
		}
		if client.Upload <= 0 {
			client.Upload = old.Upload
		}
		if client.Download <= 0 {
			client.Download = old.Download
		}
		if client.AllTime <= 0 {
			client.AllTime = old.AllTime
		}
		if client.TotalGB <= 0 {
			client.TotalGB = old.TotalGB
		}
		if client.ExpiryTime <= 0 {
			client.ExpiryTime = old.ExpiryTime
		}
		if client.Reset <= 0 {
			client.Reset = old.Reset
		}
		if client.LimitIp <= 0 {
			client.LimitIp = old.LimitIp
		}
		if client.TgId <= 0 {
			client.TgId = old.TgId
		}
		if client.LastOnline <= 0 {
			client.LastOnline = old.LastOnline
		}
		if strings.TrimSpace(client.LastIP) == "" {
			client.LastIP = old.LastIP
		}
	}
	if strings.TrimSpace(client.UUID) == "" {
		client.UUID = uuid.NewString()
	}
	if _, err := uuid.Parse(client.UUID); err != nil {
		return fmt.Errorf("invalid awg client uuid: %w", err)
	}
	client.UpdatedAt = time.Now().UnixMilli()
	if err := database.GetDB().Save(client).Error; err != nil {
		return err
	}
	server, err := s.GetServer()
	if err != nil {
		return err
	}
	if server.Enable {
		return s.applyServerConfig(server)
	}
	return nil
}

func (s *AwgService) UpdateClientByUUID(clientUUID string, client *model.AwgClient) error {
	existing, err := s.GetClientByUUID(clientUUID)
	if err != nil {
		return err
	}
	client.Id = existing.Id
	client.UUID = existing.UUID
	return s.UpdateClient(client)
}

func (s *AwgService) DeleteClient(id int) error {
	if err := database.GetDB().Delete(&model.AwgClient{}, id).Error; err != nil {
		return err
	}
	server, err := s.GetServer()
	if err != nil {
		return err
	}
	if server.Enable {
		return s.applyServerConfig(server)
	}
	return nil
}

func (s *AwgService) DeleteClientByUUID(clientUUID string) error {
	client, err := s.GetClientByUUID(clientUUID)
	if err != nil {
		return err
	}
	return s.DeleteClient(client.Id)
}

func (s *AwgService) ToggleClient(id int, enable bool) error {
	client, err := s.GetClient(id)
	if err != nil {
		return err
	}
	client.Enable = enable
	return s.UpdateClient(client)
}

func (s *AwgService) ToggleClientByUUID(clientUUID string, enable bool) error {
	client, err := s.GetClientByUUID(clientUUID)
	if err != nil {
		return err
	}
	client.Enable = enable
	return s.UpdateClient(client)
}

func (s *AwgService) ReissueClient(id int) (*model.AwgClient, error) {
	client, err := s.GetClient(id)
	if err != nil {
		return nil, err
	}
	server, err := s.GetServer()
	if err != nil {
		return nil, err
	}
	priv, pub, err := awg.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate awg client keys: %w", err)
	}
	psk, err := awg.GeneratePresharedKey()
	if err != nil {
		return nil, fmt.Errorf("generate awg psk: %w", err)
	}
	client.PrivateKey = priv
	client.PublicKey = pub
	client.PresharedKey = psk
	client.Enable = true
	if strings.TrimSpace(client.ClientAllowedIPs) == "" {
		client.ClientAllowedIPs = "0.0.0.0/0, ::/0"
	}
	if strings.TrimSpace(client.AllowedIPs) == "" || strings.TrimSpace(client.IPv4Address) == "" {
		if err := s.assignClientAddresses(server, client); err != nil {
			return nil, err
		}
	}
	if client.Jc <= 0 {
		client.Jc = server.Jc
	}
	if client.Jmin <= 0 {
		client.Jmin = server.Jmin
	}
	if client.Jmax <= 0 {
		client.Jmax = server.Jmax
	}
	if strings.TrimSpace(client.I1) == "" {
		client.I1 = server.I1
	}
	if strings.TrimSpace(client.I2) == "" {
		client.I2 = server.I2
	}
	if strings.TrimSpace(client.I3) == "" {
		client.I3 = server.I3
	}
	if strings.TrimSpace(client.I4) == "" {
		client.I4 = server.I4
	}
	if strings.TrimSpace(client.I5) == "" {
		client.I5 = server.I5
	}
	if err := s.UpdateClient(client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *AwgService) GetClientConfig(id int) (string, error) {
	client, err := s.GetClient(id)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(client.PrivateKey) == "" {
		return "", fmt.Errorf("awg client private key is not available")
	}
	server, err := s.GetServer()
	if err != nil {
		return "", err
	}
	return awg.GenerateClientConfig(server, client), nil
}

func (s *AwgService) assignClientAddresses(server *model.AwgServer, client *model.AwgClient) error {
	existing, err := s.GetClients()
	if err != nil {
		return err
	}
	usedIPv4 := make([]string, 0, len(existing))
	usedIPv6 := make([]string, 0, len(existing))
	for _, existingClient := range existing {
		if existingClient.Id == client.Id {
			continue
		}
		usedIPv4 = append(usedIPv4, existingClient.IPv4Address)
		if existingClient.IPv6Address != "" {
			usedIPv6 = append(usedIPv6, existingClient.IPv6Address)
		}
	}
	ipv4, err := awg.AllocateIPv4(server.IPv4Pool, server.IPv4Address, usedIPv4)
	if err != nil {
		return err
	}
	client.IPv4Address = ipv4
	if server.IPv6Enabled && strings.TrimSpace(server.IPv6Pool) != "" {
		ipv6, ipErr := awg.AllocateIPv6(server.IPv6Pool, server.IPv6Address, usedIPv6)
		if ipErr != nil {
			return ipErr
		}
		client.IPv6Address = ipv6
	}
	client.AllowedIPs = client.IPv4Address
	if strings.TrimSpace(client.IPv6Address) != "" {
		client.AllowedIPs += ", " + client.IPv6Address
	}
	return nil
}

func (s *AwgService) GetClientConfigByUUID(clientUUID string) (string, error) {
	client, err := s.GetClientByUUID(clientUUID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(client.PrivateKey) == "" {
		return "", fmt.Errorf("awg client private key is not available")
	}
	server, err := s.GetServer()
	if err != nil {
		return "", err
	}
	return awg.GenerateClientConfig(server, client), nil
}

func (s *AwgService) UpdateTrafficStats() {
	server, err := s.GetServer()
	if err != nil {
		return
	}
	var peers []awg.PeerRuntime
	if awg.IsInterfaceUp(server.InterfaceName) {
		peers, err = awg.RuntimePeersFromInterface(server.InterfaceName)
	} else {
		peers, err = awg.RuntimeAllPeers()
		if err == nil && len(peers) > 0 && strings.TrimSpace(peers[0].InterfaceName) != "" {
			server.InterfaceName = peers[0].InterfaceName
			server.Enable = true
			_ = database.GetDB().Save(server).Error
		}
	}
	if err != nil {
		return
	}
	var clients []model.AwgClient
	if err := database.GetDB().Find(&clients).Error; err != nil {
		return
	}
	clientByKey := make(map[string]*model.AwgClient, len(clients))
	for i := range clients {
		clientByKey[clients[i].PublicKey] = &clients[i]
	}
	_ = database.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, peer := range peers {
			client, ok := clientByKey[peer.PublicKey]
			if !ok {
				imported := runtimePeerToClient(server, peer)
				if err := tx.Create(imported).Error; err != nil {
					return err
				}
				clientByKey[peer.PublicKey] = imported
				continue
			}
			updates := map[string]any{
				"upload":   int64(peer.TransferTx),
				"download": int64(peer.TransferRx),
			}
			if peer.LatestHandshake > 0 {
				updates["last_online"] = peer.LatestHandshake * 1000
			}
			if peer.Endpoint != "" && peer.Endpoint != "(none)" {
				updates["last_ip"] = stripEndpointPort(peer.Endpoint)
			}
			tx.Model(client).Updates(updates)
		}
		return nil
	})
}

func runtimePeerToClient(server *model.AwgServer, peer awg.PeerRuntime) *model.AwgClient {
	name := runtimePeerLabel(peer.PublicKey)
	client := &model.AwgClient{
		ServerId:    server.Id,
		UUID:        uuid.NewString(),
		Name:        name,
		Email:       name,
		Enable:      true,
		Comment:     "Imported from running AmneziaWGv2 interface",
		PublicKey:   peer.PublicKey,
		Jc:          server.Jc,
		Jmin:        server.Jmin,
		Jmax:        server.Jmax,
		I1:          server.I1,
		I2:          server.I2,
		I3:          server.I3,
		I4:          server.I4,
		I5:          server.I5,
		AllowedIPs:  strings.Join(peer.AllowedIPs, ", "),
		Upload:      int64(peer.TransferTx),
		Download:    int64(peer.TransferRx),
		LastIP:      stripEndpointPort(peer.Endpoint),
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
	}
	if peer.LatestHandshake > 0 {
		client.LastOnline = peer.LatestHandshake * 1000
	}
	return client
}

func runtimePeerLabel(publicKey string) string {
	key := strings.NewReplacer("/", "", "+", "", "=", "").Replace(publicKey)
	if len(key) > 12 {
		key = key[:12]
	}
	if key == "" {
		key = uuid.NewString()[:12]
	}
	return "runtime-" + key
}

func (s *AwgService) StartIfEnabled() {
	server, err := s.GetServer()
	if err != nil || !server.Enable || awg.IsInterfaceUp(server.InterfaceName) {
		return
	}
	if err := s.applyServerConfig(server); err != nil {
		logger.Warning("restore awg v2 failed:", err)
	}
}

func (s *AwgService) applyServerConfig(server *model.AwgServer) error {
	clients, err := s.GetClients()
	if err != nil {
		return err
	}
	if err := awg.WriteServerConfig(server.InterfaceName, awg.GenerateServerConfig(server, clients)); err != nil {
		return err
	}
	if awg.IsInterfaceUp(server.InterfaceName) {
		return awg.SyncConfig(server.InterfaceName)
	}
	return awg.InterfaceUp(server.InterfaceName)
}

func stripEndpointPort(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(strings.TrimSuffix(endpoint, "]"), "[")
	if idx := strings.LastIndex(endpoint, ":"); idx > 0 {
		return endpoint[:idx]
	}
	return endpoint
}
