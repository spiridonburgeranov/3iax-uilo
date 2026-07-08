package vpnuri

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"

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

func Decode(vpnURI string) ([]byte, error) {
	encoded := strings.TrimSpace(vpnURI)
	encoded = strings.TrimPrefix(encoded, "vpn://")
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, common.NewError("decode vpn uri base64 payload:", err)
	}
	if len(raw) < 4 {
		return nil, common.NewError("invalid vpn uri payload")
	}
	reader, err := zlib.NewReader(bytes.NewReader(raw[4:]))
	if err != nil {
		return nil, common.NewError("open vpn uri zlib payload:", err)
	}
	defer reader.Close()
	decompressed := bytes.NewBuffer(nil)
	if _, err := decompressed.ReadFrom(reader); err != nil {
		return nil, common.NewError("read vpn uri zlib payload:", err)
	}
	originalLen := binary.BigEndian.Uint32(raw[:4])
	if uint32(decompressed.Len()) != originalLen {
		return nil, common.NewError("invalid vpn uri payload length")
	}
	return decompressed.Bytes(), nil
}
