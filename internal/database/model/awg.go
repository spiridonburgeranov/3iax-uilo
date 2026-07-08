package model

type AwgServer struct {
	Id                     int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Enable                 bool   `json:"enable" gorm:"default:false"`
	InterfaceName          string `json:"interfaceName" gorm:"default:'awg0'"`
	ListenPort             int    `json:"listenPort" gorm:"default:51820"`
	MTU                    int    `json:"mtu" gorm:"default:1420"`
	PrivateKey             string `json:"privateKey"`
	PublicKey              string `json:"publicKey"`
	IPv4Address            string `json:"ipv4Address" gorm:"default:'10.66.66.1/24'"`
	IPv4Pool               string `json:"ipv4Pool" gorm:"default:'10.66.66.0/24'"`
	IPv6Enabled            bool   `json:"ipv6Enabled" gorm:"default:false"`
	IPv6Address            string `json:"ipv6Address"`
	IPv6Pool               string `json:"ipv6Pool"`
	IPv6Gateway            string `json:"ipv6Gateway"`
	Jc                     int    `json:"jc" gorm:"default:4"`
	Jmin                   int    `json:"jmin" gorm:"default:50"`
	Jmax                   int    `json:"jmax" gorm:"default:1000"`
	S1                     int    `json:"s1" gorm:"default:0"`
	S2                     int    `json:"s2" gorm:"default:0"`
	H1                     int    `json:"h1" gorm:"default:1"`
	H2                     int    `json:"h2" gorm:"default:2"`
	H3                     int    `json:"h3" gorm:"default:3"`
	H4                     int    `json:"h4" gorm:"default:4"`
	DNS                    string `json:"dns" gorm:"default:'1.1.1.1,2606:4700:4700::1111'"`
	ExternalInterface      string `json:"externalInterface"`
	IPv6ExternalInterface  string `json:"ipv6ExternalInterface"`
	PostUp                 string `json:"postUp"`
	PostDown               string `json:"postDown"`
	Endpoint               string `json:"endpoint"`
	TrafficReset           string `json:"trafficReset" gorm:"default:'never'"`
	CreatedAt              int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt              int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

type AwgClient struct {
	Id                  int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ServerId            int    `json:"serverId" gorm:"index"`
	UUID                string `json:"uuid" gorm:"index"`
	Name                string `json:"name"`
	Email               string `json:"email" gorm:"uniqueIndex"`
	Enable              bool   `json:"enable" gorm:"default:true"`
	Comment             string `json:"comment"`
	PrivateKey          string `json:"privateKey"`
	PublicKey           string `json:"publicKey"`
	PresharedKey        string `json:"presharedKey"`
	IPv4Address         string `json:"ipv4Address"`
	IPv6Address         string `json:"ipv6Address"`
	AllowedIPs          string `json:"allowedIPs"`
	ClientAllowedIPs    string `json:"clientAllowedIPs" gorm:"default:'0.0.0.0/0,::/0'"`
	ForwardedPorts      string `json:"forwardedPorts"`
	PersistentKeepalive int    `json:"persistentKeepalive" gorm:"default:25"`
	Upload              int64  `json:"upload" gorm:"default:0"`
	Download            int64  `json:"download" gorm:"default:0"`
	TotalGB             int64  `json:"totalGB" gorm:"default:0"`
	AllTime             int64  `json:"allTime" gorm:"default:0"`
	ExpiryTime          int64  `json:"expiryTime" gorm:"default:0"`
	Reset               int    `json:"reset" gorm:"default:0"`
	LimitIp             int    `json:"limitIp" gorm:"default:0"`
	TgId                int64  `json:"tgId" gorm:"default:0"`
	LastOnline          int64  `json:"lastOnline" gorm:"default:0"`
	LastIP              string `json:"lastIp"`
	CreatedAt           int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt           int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}
