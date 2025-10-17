package processor

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestParseScanEnvelopeV1(t *testing.T) {
	payload := []byte(`{
		"ip": "192.0.2.1",
		"port": 443,
		"service": "HTTPS",
		"timestamp": 1700000000,
		"data_version": 1,
		"data": {
			"response_bytes_utf8": "` + base64.StdEncoding.EncodeToString([]byte("hello world")) + `"
		}
	}`)

	scan, err := ParseScanEnvelope(payload)
	if err != nil {
		t.Fatalf("ParseScanEnvelope returned error: %v", err)
	}

	if scan.Response != "hello world" {
		t.Fatalf("expected response %q, got %q", "hello world", scan.Response)
	}
	if !scan.ObservedAt.Equal(time.Unix(873237600, 0).UTC()) {
		t.Fatalf("unexpected observed timestamp: %v", scan.ObservedAt)
	}
}

func TestParseScanEnvelopeV2(t *testing.T) {
	payload := []byte(`{
		"ip": "192.0.2.1",
		"port": 80,
		"service": "HTTP",
		"timestamp": 1700000001,
		"data_version": 2,
		"data": {
			"response_str": "hello world"
		}
	}`)

	scan, err := ParseScanEnvelope(payload)
	if err != nil {
		t.Fatalf("ParseScanEnvelope returned error: %v", err)
	}
	if scan.Response != "hello world" {
		t.Fatalf("expected response %q, got %q", "hello world", scan.Response)
	}
}

func TestParseScanEnvelopeUnknownVersion(t *testing.T) {
	payload := []byte(`{
		"ip": "192.0.2.1",
		"port": 80,
		"service": "HTTP",
		"timestamp": 1700000001,
		"data_version": 99,
		"data": {}
	}`)

	if _, err := ParseScanEnvelope(payload); err == nil {
		t.Fatal("expected error for unknown data_version")
	}
}
