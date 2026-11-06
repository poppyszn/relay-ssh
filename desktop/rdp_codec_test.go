package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestRDCleanPathCodecRoundTripRequest(t *testing.T) {
	req := RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		Destination:       "192.168.1.50:3389",
		ProxyAuth:         "test-session-token-1234",
		X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	der, err := EncodeRDCleanPath(req)
	if err != nil {
		t.Fatalf("EncodeRDCleanPath: %v", err)
	}

	decoded, err := DecodeRDCleanPath(der)
	if err != nil {
		t.Fatalf("DecodeRDCleanPath: %v", err)
	}

	if decoded.Version != RDCleanPathVersion1 {
		t.Errorf("version = %d, want %d", decoded.Version, RDCleanPathVersion1)
	}
	if decoded.Destination != req.Destination {
		t.Errorf("destination = %q, want %q", decoded.Destination, req.Destination)
	}
	if decoded.ProxyAuth != req.ProxyAuth {
		t.Errorf("proxy_auth = %q, want %q", decoded.ProxyAuth, req.ProxyAuth)
	}
	if !bytes.Equal(decoded.X224ConnectionPdu, req.X224ConnectionPdu) {
		t.Errorf("x224 = %x, want %x", decoded.X224ConnectionPdu, req.X224ConnectionPdu)
	}
}

func TestRDCleanPathCodecRoundTripResponse(t *testing.T) {
	cert1 := []byte{0x30, 0x82, 0x01, 0x0a}
	cert2 := []byte{0x30, 0x82, 0x02, 0x0b}
	x224CC := []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00}

	resp := NewRDCleanPathResponsePdu("10.0.0.1:3389", x224CC, [][]byte{cert1, cert2})
	der, err := EncodeRDCleanPath(resp)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, err := DecodeRDCleanPath(der)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Version != RDCleanPathVersion1 {
		t.Errorf("version = %d, want %d", decoded.Version, RDCleanPathVersion1)
	}
	if decoded.ServerAddr != "10.0.0.1:3389" {
		t.Errorf("server_addr = %q, want 10.0.0.1:3389", decoded.ServerAddr)
	}
	if !bytes.Equal(decoded.X224ConnectionPdu, x224CC) {
		t.Errorf("x224 mismatch: got %x, want %x", decoded.X224ConnectionPdu, x224CC)
	}
	if len(decoded.ServerCertChain) != 2 {
		t.Fatalf("cert chain len = %d, want 2", len(decoded.ServerCertChain))
	}
	if !bytes.Equal(decoded.ServerCertChain[0], cert1) || !bytes.Equal(decoded.ServerCertChain[1], cert2) {
		t.Errorf("cert chain contents mismatch")
	}
}

func TestRDCleanPathCodecRoundTripError(t *testing.T) {
	errPdu := NewRDCleanPathErrorPdu(TargetUnreachableErrorCode)
	der, err := EncodeRDCleanPath(errPdu)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	decoded, err := DecodeRDCleanPath(der)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if decoded.Version != RDCleanPathVersion1 {
		t.Errorf("version = %d, want %d", decoded.Version, RDCleanPathVersion1)
	}
	if decoded.Error == nil || decoded.Error.ErrorCode != TargetUnreachableErrorCode {
		t.Errorf("error code = %v, want %d", decoded.Error, TargetUnreachableErrorCode)
	}
}

func TestRDCleanPathGoldenTagVerification(t *testing.T) {
	// Golden check: verify explicit context-specific tags in encoded bytes.
	// [0] EXPLICIT INTEGER (version 3390 = 0x0D3E)
	pdu := RDCleanPathPdu{
		Version:     RDCleanPathVersion1,
		Destination: "host",
	}
	der, err := EncodeRDCleanPath(pdu)
	if err != nil {
		t.Fatal(err)
	}

	// der[0] should be 0x30 (SEQUENCE tag)
	if der[0] != 0x30 {
		t.Fatalf("der[0] = 0x%02x, want 0x30 (SEQUENCE)", der[0])
	}

	// [0] tag is 0xA0 (context-specific, constructed, tag 0)
	// Body of tag 0 is INTEGER: 0x02 0x02 0x0D 0x3E
	expectedVersionBytes := []byte{0xA0, 0x04, 0x02, 0x02, 0x0D, 0x3E}
	if !bytes.Contains(der, expectedVersionBytes) {
		t.Errorf("encoded DER missing [0] EXPLICIT version 3390, got: %x", der)
	}

	// [2] tag is 0xA2 (context-specific, constructed, tag 2)
	// Body of tag 2 is UTF8String: 0x0C 0x04 'h' 'o' 's' 't'
	expectedDestBytes := []byte{0xA2, 0x06, 0x0C, 0x04, 'h', 'o', 's', 't'}
	if !bytes.Contains(der, expectedDestBytes) {
		t.Errorf("encoded DER missing [2] EXPLICIT destination, got: %x", der)
	}
}

// Bytes produced by IronRDP's own ironrdp-rdcleanpath crate (npm-iron-remote-desktop-rdp-v0.7.0)
// for new_request(x224, "10.0.2.50:3389", "tok-123", Some("pcb")). This is what the real browser
// client sends, so decoding it proves our codec reads the official wire format.
const ironrdpRequestHex = "3039a00402020d3ea2100c0e31302e302e322e35303a33333839a3090c07746f6b2d313233a5050c03706362a60d040b0300000b06e00000000000"

func TestDecodesARequestWrittenByIronRDP(t *testing.T) {
	raw, err := hex.DecodeString(ironrdpRequestHex)
	if err != nil {
		t.Fatal(err)
	}
	pdu, err := DecodeRDCleanPath(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pdu.Version != RDCleanPathVersion1 || pdu.Destination != "10.0.2.50:3389" || pdu.ProxyAuth != "tok-123" || pdu.PreconnectionBlob != "pcb" {
		t.Fatalf("fields wrong: %+v", pdu)
	}
	if hex.EncodeToString(pdu.X224ConnectionPdu) != "0300000b06e00000000000" {
		t.Fatalf("x224 = %x", pdu.X224ConnectionPdu)
	}
}

// What our gateway sends back. IronRDP's ironrdp-rdcleanpath crate decoded these exact bytes (version 3390,
// server address, X.224 confirm and a two-certificate chain; and a general error) and accepted them as a valid
// response and error, so the browser client will too.
const (
	ironrdpAcceptedResponseHex = "3034a00402020d3ea60d040b0300000b06d00000000000a70b3009040301020304020405a9100c0e31302e302e322e35303a33333839"
	ironrdpAcceptedErrorHex    = "300fa00402020d3ea1073005a003020101"
)

func TestEncodesWhatIronRDPAccepts(t *testing.T) {
	resp, err := EncodeRDCleanPath(NewRDCleanPathResponsePdu("10.0.2.50:3389", []byte{3, 0, 0, 11, 6, 0xd0, 0, 0, 0, 0, 0}, [][]byte{{1, 2, 3}, {4, 5}}))
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(resp); got != ironrdpAcceptedResponseHex {
		t.Fatalf("response bytes changed:\n got %s\nwant %s", got, ironrdpAcceptedResponseHex)
	}
	e, err := EncodeRDCleanPath(NewRDCleanPathErrorPdu(GeneralErrorCode))
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(e); got != ironrdpAcceptedErrorHex {
		t.Fatalf("error bytes changed:\n got %s\nwant %s", got, ironrdpAcceptedErrorHex)
	}
}
