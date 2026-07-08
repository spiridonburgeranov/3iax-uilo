package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/awg"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	"gorm.io/gorm"
)

type AwgInboundService struct{}

type AwgDiscoveredInterface struct {
	Name        string `json:"name"`
	ConfigPath  string `json:"configPath"`
	ListenPort  int    `json:"listenPort"`
	Address     string `json:"address"`
	PeerCount   int    `json:"peerCount"`
	Running     bool   `json:"running"`
	Imported    bool   `json:"imported"`
	InboundID   int    `json:"inboundId,omitempty"`
	InboundTag  string `json:"inboundTag,omitempty"`
	InboundNote string `json:"inboundRemark,omitempty"`
}

type AwgImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

type AwgInboundRuntime struct {
	InboundID     int               `json:"inboundId"`
	Remark        string            `json:"remark"`
	Tag           string            `json:"tag"`
	Port          int               `json:"port"`
	Enable        bool              `json:"enable"`
	InterfaceName string            `json:"interfaceName"`
	Running       bool              `json:"running"`
	PeerCount     int               `json:"peerCount"`
	OnlineCount   int               `json:"onlineCount"`
	Peers         []awg.PeerRuntime `json:"peers"`
}

func (s *AwgInboundService) Apply(inbound *model.Inbound) error {
	if inbound == nil || inbound.NodeID != nil || inbound.Protocol != model.AmneziaWG {
		return nil
	}
	if !inbound.Enable {
		return awg.DisableInbound(inbound)
	}
	return awg.ApplyInbound(inbound)
}

func (s *AwgInboundService) Disable(inbound *model.Inbound) error {
	if inbound == nil || inbound.NodeID != nil || inbound.Protocol != model.AmneziaWG {
		return nil
	}
	return awg.DisableInbound(inbound)
}

func (s *AwgInboundService) Remove(inbound *model.Inbound) error {
	if inbound == nil || inbound.NodeID != nil || inbound.Protocol != model.AmneziaWG {
		return nil
	}
	_ = awg.DisableInbound(inbound)
	return awg.RemoveConfig(inbound)
}

func (s *AwgInboundService) RestoreAll() {
	if !awg.IsInstalled() {
		return
	}
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("protocol = ? AND node_id IS NULL AND enable = ?", model.AmneziaWG, true).
		Find(&inbounds).Error; err != nil {
		logger.Warning("awg restore list failed:", err)
		return
	}
	for i := range inbounds {
		if awg.IsInboundUp(&inbounds[i]) {
			continue
		}
		if err := s.Apply(&inbounds[i]); err != nil {
			logger.Warning("awg restore inbound", inbounds[i].Tag, "failed:", err)
		}
	}
}

func (s *AwgInboundService) StartupScanAndImport() {
	if !awg.IsInstalled() {
		return
	}
	result, scanErr := s.ImportDiscovered(false, nil)
	if scanErr != nil {
		logger.Warning("awg startup scan failed:", scanErr)
	}
	if result.Imported > 0 {
		logger.Infof("awg startup scan imported %d interface(s)", result.Imported)
	}
	for _, entry := range result.Errors {
		logger.Warning("awg startup import:", entry)
	}
	s.RestoreAll()
}

type AwgProvisionResult struct {
	Remark        string         `json:"remark"`
	Port          int            `json:"port"`
	Enable        bool           `json:"enable"`
	Tag           string         `json:"tag"`
	PublicKey     string         `json:"publicKey"`
	InterfaceName string         `json:"interfaceName"`
	ConfigPath    string         `json:"configPath"`
	Settings      map[string]any `json:"settings"`
}

func (s *AwgInboundService) ProvisionNew() (*AwgProvisionResult, error) {
	if !awg.IsInstalled() {
		return nil, fmt.Errorf("amneziawg runtime is not installed (missing awg/awg-quick)")
	}
	snapshot := s.buildResourceSnapshot()
	plan, err := awg.BuildProvisionPlan(snapshot)
	if err != nil {
		return nil, err
	}
	settings := map[string]any{
		"secretKey":    plan.PrivateKey,
		"address":      plan.ServerAddress,
		"dns":          plan.DNS,
		"awgInterface": plan.InterfaceName,
		"mtu":          plan.MTU,
		"jc":           plan.Jc,
		"jmin":         plan.Jmin,
		"jmax":         plan.Jmax,
		"s1":           plan.S1,
		"s2":           plan.S2,
		"s3":           plan.S3,
		"s4":           plan.S4,
		"h1":           plan.H1,
		"h2":           plan.H2,
		"h3":           plan.H3,
		"h4":           plan.H4,
		"i1":           "",
		"i2":           "",
		"i3":           "",
		"i4":           "",
		"i5":           "",
		"postUp":       "",
		"postDown":     "",
		"clients":      []any{},
		"peers":        []any{},
	}
	return &AwgProvisionResult{
		Remark:        "AmneziaWG " + plan.InterfaceName,
		Port:          plan.ListenPort,
		Enable:        true,
		Tag:           "inbound-" + plan.InterfaceName,
		PublicKey:     plan.PublicKey,
		InterfaceName: plan.InterfaceName,
		ConfigPath:    awg.ConfigPathForInterface(plan.InterfaceName),
		Settings:      settings,
	}, nil
}

func (s *AwgInboundService) buildResourceSnapshot() awg.ResourceSnapshot {
	names := make([]string, 0, 16)
	ports := make([]int, 0, 16)
	subnets := make([]string, 0, 16)
	if discovered, err := awg.DiscoverInterfaces(); err == nil {
		for _, entry := range discovered {
			names = append(names, entry.Name)
			if entry.ListenPort > 0 {
				ports = append(ports, entry.ListenPort)
			}
			if port := awg.ParseInterfacePort(entry.Name); port > 0 {
				ports = append(ports, port)
			}
			if strings.TrimSpace(entry.Address) != "" {
				subnets = append(subnets, entry.Address)
			}
		}
	}
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("protocol = ? AND node_id IS NULL", model.AmneziaWG).Find(&inbounds).Error; err == nil {
		for _, inbound := range inbounds {
			iface := awg.InterfaceName(&inbound)
			names = append(names, iface)
			if inbound.Port > 0 {
				ports = append(ports, inbound.Port)
			}
			if port := awg.ParseInterfacePort(iface); port > 0 {
				ports = append(ports, port)
			}
			var settings map[string]any
			if err := json.Unmarshal([]byte(inbound.Settings), &settings); err == nil {
				if address, ok := settings["address"].(string); ok && strings.TrimSpace(address) != "" {
					subnets = append(subnets, address)
				}
			}
		}
	}
	return awg.SnapshotFromNamesPortsSubnets(names, ports, subnets)
}

type AwgInboundTemplate struct {
	Remark   string         `json:"remark"`
	Port     int            `json:"port"`
	Enable   bool           `json:"enable"`
	Tag      string         `json:"tag"`
	Settings map[string]any `json:"settings"`
}

func (s *AwgInboundService) TemplateFromInterface(name string) (*AwgInboundTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("interface name is required")
	}
	discovered, err := awg.DiscoverInterfaces()
	if err != nil {
		return nil, err
	}
	for _, entry := range discovered {
		if entry.Name != name {
			continue
		}
		inbound, ierr := s.buildInboundFromParsed(entry)
		if ierr != nil {
			return nil, ierr
		}
		var settings map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			return nil, err
		}
		return &AwgInboundTemplate{
			Remark:   inbound.Remark,
			Port:     inbound.Port,
			Enable:   inbound.Enable,
			Tag:      inbound.Tag,
			Settings: settings,
		}, nil
	}
	return nil, fmt.Errorf("interface %s not found", name)
}

func (s *AwgInboundService) ListDiscovered() ([]AwgDiscoveredInterface, error) {
	discovered, err := awg.DiscoverInterfaces()
	if err != nil {
		return nil, err
	}
	known := s.knownInterfaceMap()
	out := make([]AwgDiscoveredInterface, 0, len(discovered))
	for _, entry := range discovered {
		item := AwgDiscoveredInterface{
			Name:       entry.Name,
			ConfigPath: entry.ConfigPath,
			ListenPort: entry.ListenPort,
			Address:    entry.Address,
			PeerCount:  len(entry.Peers),
			Running:    entry.Running || awg.IsInterfaceUp(entry.Name),
		}
		if inbound, ok := known[entry.Name]; ok {
			item.Imported = true
			item.InboundID = inbound.Id
			item.InboundTag = inbound.Tag
			item.InboundNote = inbound.Remark
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AwgInboundService) ImportDiscovered(force bool, names []string) (AwgImportResult, error) {
	result := AwgImportResult{}
	discovered, err := awg.DiscoverInterfaces()
	if err != nil {
		return result, err
	}
	only := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			only[name] = struct{}{}
		}
	}
	known := s.knownInterfaceMap()
	inboundSvc := InboundService{}
	for _, entry := range discovered {
		if len(only) > 0 {
			if _, ok := only[entry.Name]; !ok {
				continue
			}
		}
		if _, ok := known[entry.Name]; ok {
			result.Skipped++
			continue
		}
		if !force && !entry.Running && strings.TrimSpace(entry.PrivateKey) == "" {
			result.Skipped++
			continue
		}
		inbound, ierr := s.buildInboundFromParsed(entry)
		if ierr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Name, ierr))
			continue
		}
		if _, _, ierr = inboundSvc.addInbound(inbound, inboundPersistOptions{skipRuntimeApply: true}); ierr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Name, ierr))
			continue
		}
		result.Imported++
	}
	return result, nil
}

func (s *AwgInboundService) ListRuntime() ([]AwgInboundRuntime, error) {
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("protocol = ? AND node_id IS NULL", model.AmneziaWG).
		Order("id asc").Find(&inbounds).Error; err != nil {
		return nil, err
	}
	out := make([]AwgInboundRuntime, 0, len(inbounds))
	for i := range inbounds {
		ib := inbounds[i]
		iface := awg.InterfaceName(&ib)
		runtime := AwgInboundRuntime{
			InboundID:     ib.Id,
			Remark:        ib.Remark,
			Tag:           ib.Tag,
			Port:          ib.Port,
			Enable:        ib.Enable,
			InterfaceName: iface,
			Running:       awg.IsInboundUp(&ib),
		}
		peers, perr := awg.RuntimePeers(&ib)
		if perr == nil {
			runtime.Peers = peers
			runtime.PeerCount = len(peers)
			for _, peer := range peers {
				if peer.Online {
					runtime.OnlineCount++
				}
			}
		}
		out = append(out, runtime)
	}
	return out, nil
}

func (s *AwgInboundService) ToggleAll(enable bool) error {
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("protocol = ? AND node_id IS NULL", model.AmneziaWG).Find(&inbounds).Error; err != nil {
		return err
	}
	for i := range inbounds {
		inbounds[i].Enable = enable
		if err := db.Model(&inbounds[i]).Update("enable", enable).Error; err != nil {
			return err
		}
		if enable {
			if err := s.Apply(&inbounds[i]); err != nil {
				return err
			}
			continue
		}
		if err := s.Disable(&inbounds[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *AwgInboundService) UpdateTrafficStats() {
	if !awg.IsInstalled() {
		return
	}
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("protocol = ? AND node_id IS NULL", model.AmneziaWG).Find(&inbounds).Error; err != nil {
		return
	}
	inboundSvc := InboundService{}
	for i := range inbounds {
		ib := inbounds[i]
		peers, err := awg.RuntimePeers(&ib)
		if err != nil {
			continue
		}
		clients, err := inboundSvc.GetClients(&ib)
		if err != nil {
			continue
		}
		emailByKey := make(map[string]string, len(clients))
		for _, client := range clients {
			key := strings.TrimSpace(client.PublicKey)
			if key == "" && strings.TrimSpace(client.PrivateKey) != "" {
				if derived, derr := wgPublicFromPrivate(client.PrivateKey); derr == nil {
					key = derived
				}
			}
			if key != "" && strings.TrimSpace(client.Email) != "" {
				emailByKey[key] = client.Email
			}
		}
		_ = db.Transaction(func(tx *gorm.DB) error {
			for _, peer := range peers {
				email := emailByKey[peer.PublicKey]
				if email == "" {
					continue
				}
				updates := map[string]any{
					"up":   int64(peer.TransferTx),
					"down": int64(peer.TransferRx),
				}
				if peer.LatestHandshake > 0 {
					updates["last_online"] = peer.LatestHandshake * 1000
				}
				tx.Model(&xray.ClientTraffic{}).Where("email = ?", email).Updates(updates)
			}
			return nil
		})
	}
}

func (s *AwgInboundService) ClientConfig(inbound *model.Inbound, client *model.Client, endpoint string) (string, error) {
	if inbound == nil || client == nil {
		return "", fmt.Errorf("inbound and client are required")
	}
	return awg.GenerateClientConfig(inbound, awg.ClientConfigInput{
		PrivateKey:       client.PrivateKey,
		PublicKey:        client.PublicKey,
		AllowedIPs:       client.AllowedIPs,
		PreSharedKey:     client.PreSharedKey,
		KeepAlive:        client.KeepAlive,
		ClientAllowedIPs: "0.0.0.0/0, ::/0",
	}, endpoint)
}

func (s *AwgInboundService) knownInterfaceMap() map[string]model.Inbound {
	db := database.GetDB()
	var inbounds []model.Inbound
	_ = db.Where("protocol = ? AND node_id IS NULL", model.AmneziaWG).Find(&inbounds).Error
	out := make(map[string]model.Inbound, len(inbounds))
	for _, inbound := range inbounds {
		out[awg.InterfaceName(&inbound)] = inbound
	}
	return out
}

func (s *AwgInboundService) buildInboundFromParsed(entry awg.ParsedInterface) (*model.Inbound, error) {
	userID, err := firstPanelUserID()
	if err != nil {
		return nil, err
	}
	port := entry.ListenPort
	if port <= 0 {
		port = 51820
	}
	address := strings.TrimSpace(entry.Address)
	if address == "" {
		address = "10.66.66.1/24"
	}
	dns := strings.TrimSpace(entry.DNS)
	if dns == "" {
		dns = "1.1.1.1,2606:4700:4700::1111"
	}
	clients := make([]model.Client, 0, len(entry.Peers))
	now := time.Now().UnixMilli()
	seenEmails := map[string]struct{}{}
	for _, peer := range entry.Peers {
		if strings.TrimSpace(peer.PublicKey) == "" {
			continue
		}
		email := importScopedEmail(entry.Name, peer.Name, peer.PublicKey, seenEmails)
		clients = append(clients, model.Client{
			ID:           uuid.NewString(),
			Email:        email,
			Enable:       true,
			PublicKey:    peer.PublicKey,
			PreSharedKey: peer.PresharedKey,
			AllowedIPs:   peer.AllowedIPs,
			KeepAlive:    peer.KeepAlive,
			Comment:      "Imported from AmneziaWG interface " + entry.Name,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	settings := map[string]any{
		"secretKey":    entry.PrivateKey,
		"address":      address,
		"dns":          dns,
		"awgInterface": entry.Name,
		"mtu":          entry.MTU,
		"jc":           entry.Jc,
		"jmin":         entry.Jmin,
		"jmax":         entry.Jmax,
		"s1":           entry.S1,
		"s2":           entry.S2,
		"s3":           entry.S3,
		"s4":           entry.S4,
		"h1":           entry.H1,
		"h2":           entry.H2,
		"h3":           entry.H3,
		"h4":           entry.H4,
		"i1":           entry.I1,
		"i2":           entry.I2,
		"i3":           entry.I3,
		"i4":           entry.I4,
		"i5":           entry.I5,
		"postUp":       entry.PostUp,
		"postDown":     entry.PostDown,
		"clients":      clients,
		"peers":        []any{},
	}
	if strings.TrimSpace(entry.PrivateKey) == "" {
		return nil, fmt.Errorf("missing private key for %s; set XUI_AWG_CONFIG_DIR or ensure awg showconf works", entry.Name)
	}
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	remark := "AmneziaWG " + entry.Name
	if entry.Running {
		remark += " (running)"
	}
	return &model.Inbound{
		UserId:   userID,
		Remark:   remark,
		Enable:   entry.Running || awg.IsInterfaceUp(entry.Name),
		Port:     port,
		Protocol: model.AmneziaWG,
		Settings: string(settingsJSON),
		Tag:      "inbound-" + entry.Name,
	}, nil
}

func firstPanelUserID() (int, error) {
	db := database.GetDB()
	var user model.User
	if err := db.Order("id asc").First(&user).Error; err != nil {
		return 0, err
	}
	return user.Id, nil
}

func wgPublicFromPrivate(privateKey string) (string, error) {
	return wireguard.PublicKeyFromPrivate(privateKey)
}

func importScopedEmail(iface, peerName, publicKey string, seen map[string]struct{}) string {
	base := strings.TrimSpace(peerName)
	if base == "" {
		base = runtimePeerLabel(publicKey)
	}
	candidate := base
	if strings.TrimSpace(iface) != "" {
		candidate = iface + "/" + base
	}
	key := strings.ToLower(candidate)
	for {
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			return candidate
		}
		candidate = candidate + "-2"
		key = strings.ToLower(candidate)
	}
}
