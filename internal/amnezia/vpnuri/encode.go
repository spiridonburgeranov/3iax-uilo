package vpnuri

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

func Encode(outer any) (string, error) {
	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return "", common.NewError("marshal amnezia vpn uri json:", err)
	}
	var compressed bytes.Buffer
	writer, err := zlib.NewWriterLevel(&compressed, zlib.DefaultCompression)
	if err != nil {
		return "", err
	}
	if _, err := writer.Write(outerJSON); err != nil {
		_ = writer.Close()
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	payload := make([]byte, 4+compressed.Len())
	binary.BigEndian.PutUint32(payload[0:4], uint32(len(outerJSON)))
	copy(payload[4:], compressed.Bytes())
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return "vpn://" + encoded, nil
}
