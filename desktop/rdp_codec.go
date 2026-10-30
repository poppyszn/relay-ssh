package main

import (
	"encoding/asn1"
	"fmt"
)

// RDCleanPath protocol constants.
const (
	RDCleanPathVersion1 = 3390

	// Error codes defined by RDCleanPath.
	GeneralErrorCode             = 1
	NegotiationErrorCode         = 2
	ExpiredErrorCode             = 100
	NotYetValidErrorCode         = 101
	RevokedErrorCode             = 102
	TargetUnreachableErrorCode   = 103
	DNSResolutionFailedErrorCode = 104
	TLSHandshakeFailedErrorCode  = 105
)

// RDCleanPathErr is the error structure inside an RDCleanPath PDU.
type RDCleanPathErr struct {
	ErrorCode      int `asn1:"explicit,tag:0"`
	HTTPStatusCode int `asn1:"explicit,optional,tag:1"`
	WSALastError   int `asn1:"explicit,optional,tag:2"`
	TLSAlertCode   int `asn1:"explicit,optional,tag:3"`
}

// RDCleanPathPdu represents the ASN.1 DER sequence exchanged in RDCleanPath handshakes.
type RDCleanPathPdu struct {
	Version           int             `asn1:"explicit,tag:0"`
	Error             *RDCleanPathErr `asn1:"explicit,optional,tag:1"`
	Destination       string          `asn1:"explicit,optional,utf8,tag:2"`
	ProxyAuth         string          `asn1:"explicit,optional,utf8,tag:3"`
	ServerAuth        string          `asn1:"explicit,optional,utf8,tag:4"`
	PreconnectionBlob string          `asn1:"explicit,optional,utf8,tag:5"`
	X224ConnectionPdu []byte          `asn1:"explicit,optional,tag:6"`
	ServerCertChain   [][]byte        `asn1:"explicit,optional,tag:7"`
	ServerAddr        string          `asn1:"explicit,optional,utf8,tag:9"`
}

type rawRDCleanPathPdu struct {
	Version           int           `asn1:"explicit,tag:0"`
	Error             asn1.RawValue `asn1:"optional,tag:1"`
	Destination       string        `asn1:"explicit,optional,utf8,tag:2"`
	ProxyAuth         string        `asn1:"explicit,optional,utf8,tag:3"`
	ServerAuth        string        `asn1:"explicit,optional,utf8,tag:4"`
	PreconnectionBlob string        `asn1:"explicit,optional,utf8,tag:5"`
	X224ConnectionPdu []byte        `asn1:"explicit,optional,tag:6"`
	ServerCertChain   [][]byte      `asn1:"explicit,optional,tag:7"`
	ServerAddr        string        `asn1:"explicit,optional,utf8,tag:9"`
}

// EncodeRDCleanPath encodes an RDCleanPathPdu into ASN.1 DER.
func EncodeRDCleanPath(pdu RDCleanPathPdu) ([]byte, error) {
	if pdu.Version == 0 {
		pdu.Version = RDCleanPathVersion1
	}
	raw := rawRDCleanPathPdu{
		Version:           pdu.Version,
		Destination:       pdu.Destination,
		ProxyAuth:         pdu.ProxyAuth,
		ServerAuth:        pdu.ServerAuth,
		PreconnectionBlob: pdu.PreconnectionBlob,
		X224ConnectionPdu: pdu.X224ConnectionPdu,
		ServerCertChain:   pdu.ServerCertChain,
		ServerAddr:        pdu.ServerAddr,
	}
	if pdu.Error != nil {
		errBytes, err := asn1.Marshal(*pdu.Error)
		if err != nil {
			return nil, fmt.Errorf("marshaling error pdu: %w", err)
		}
		raw.Error = asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        1,
			IsCompound: true,
			Bytes:      errBytes,
		}
	}
	return asn1.Marshal(raw)
}

// DecodeRDCleanPath decodes an ASN.1 DER buffer into an RDCleanPathPdu.
func DecodeRDCleanPath(data []byte) (RDCleanPathPdu, error) {
	var raw rawRDCleanPathPdu
	rest, err := asn1.Unmarshal(data, &raw)
	if err != nil {
		return RDCleanPathPdu{}, fmt.Errorf("decoding rdcleanpath pdu: %w", err)
	}
	if len(rest) > 0 {
		return RDCleanPathPdu{}, fmt.Errorf("unexpected trailing %d bytes in rdcleanpath pdu", len(rest))
	}
	pdu := RDCleanPathPdu{
		Version:           raw.Version,
		Destination:       raw.Destination,
		ProxyAuth:         raw.ProxyAuth,
		ServerAuth:        raw.ServerAuth,
		PreconnectionBlob: raw.PreconnectionBlob,
		X224ConnectionPdu: raw.X224ConnectionPdu,
		ServerCertChain:   raw.ServerCertChain,
		ServerAddr:        raw.ServerAddr,
	}
	if raw.Error.Class == asn1.ClassContextSpecific && raw.Error.Tag == 1 && len(raw.Error.Bytes) > 0 {
		var errObj RDCleanPathErr
		if _, err := asn1.Unmarshal(raw.Error.Bytes, &errObj); err != nil {
			return RDCleanPathPdu{}, fmt.Errorf("decoding rdcleanpath error: %w", err)
		}
		pdu.Error = &errObj
	}
	return pdu, nil
}

// NewRDCleanPathErrorPdu creates an error PDU with the given error code.
func NewRDCleanPathErrorPdu(code int) RDCleanPathPdu {
	return RDCleanPathPdu{
		Version: RDCleanPathVersion1,
		Error: &RDCleanPathErr{
			ErrorCode: code,
		},
	}
}

// NewRDCleanPathResponsePdu creates a successful gateway response PDU.
func NewRDCleanPathResponsePdu(serverAddr string, x224 []byte, certChain [][]byte) RDCleanPathPdu {
	return RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		ServerAddr:        serverAddr,
		X224ConnectionPdu: x224,
		ServerCertChain:   certChain,
	}
}
