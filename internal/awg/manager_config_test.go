package awg

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestBuildConfigOmitsDNSForServerRuntime(t *testing.T) {
	t.Helper()
	inbound := &model.Inbound{
		Id:   1,
		Port: 51821,
		Settings: `{
			"secretKey":"cGJ4VGVzdEtleVRlc3RLZXlUZXN0S2V5VGVzdEtleVRlc3Q=",
			"address":"10.66.67.1/24",
			"dns":"1.1.1.1,2606:4700:4700::1111",
			"mtu":1420,
			"jc":4,"jmin":64,"jmax":256,
			"s1":15,"s2":25,"s3":35,"s4":15,
			"h1":"11111111-2222-3333-4444-555555555555",
			"h2":"22222222-3333-4444-5555-666666666666",
			"h3":"33333333-4444-5555-6666-777777777777",
			"h4":"44444444-5555-6666-7777-888888888888",
			"i1":"<b 0x01>","i2":"<b 0x02>","i3":"<b 0x03>","i4":"<b 0x04>","i5":"<b 0x05>",
			"clients":[]
		}`,
	}
	cfg, err := buildConfig(inbound)
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	if strings.Contains(cfg, "DNS = ") {
		t.Fatalf("server runtime config must not include DNS (awg-quick needs resolvconf): %q", cfg)
	}
}
