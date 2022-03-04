package archival

//
// This file defines the "flat" data format. We use this data format
// as a simpler to process data format used to exchange information
// between the probe and the websteps test helper.
//
// Incidentally, this is also the internal data format that we use
// to store and process measurement results.
//
// This data format is different from the "archival" data format, i.e.,
// the data format used to serialize and submit measurements.
//

import (
	"net/http"
	"time"
)

// FlatDNSLookupEvent contains the results of a DNS lookup.
type FlatDNSLookupEvent struct {
	ALPNs           []string
	Addresses       []string
	Domain          string
	Failure         FlatFailure
	Finished        time.Time
	LookupType      DNSLookupType
	ResolverAddress string
	ResolverNetwork string
	Started         time.Time
}

// FlatDNSRoundTripEvent contains the result of a DNS round trip.
type FlatDNSRoundTripEvent struct {
	Address  string
	Failure  FlatFailure
	Finished time.Time
	Network  string
	Query    []byte
	Reply    []byte
	Started  time.Time
}

// FlatFailure is the flat data format representation of failure.
type FlatFailure string

// NewFlatFailure constructs a new FlatFailure from an error.
func NewFlatFailure(err error) FlatFailure {
	if err != nil {
		return FlatFailure(err.Error())
	}
	return ""
}

// ToArchivalFailure converts from FlatFailure to ArchivalFailure.
func (ff FlatFailure) ToArchivalFailure() *string {
	if ff != "" {
		s := string(ff)
		return &s
	}
	return nil
}

// IsSuccess returns true if there is no failure, false otherwise.
func (ff FlatFailure) IsSuccess() bool {
	return ff == ""
}

// FlatHTTPRoundTripEvent contains an HTTP round trip.
type FlatHTTPRoundTripEvent struct {
	Failure                 FlatFailure
	Finished                time.Time
	Method                  string
	RequestHeaders          http.Header
	ResponseBody            []byte
	ResponseBodyIsTruncated bool
	ResponseBodyLength      int64
	ResponseHeaders         http.Header
	Started                 time.Time
	StatusCode              int64
	Transport               string
	URL                     string
}

// FlatNetworkEvent contains a network event. This kind of events
// are generated by Dialer, QUICDialer, Conn, QUICConn.
type FlatNetworkEvent struct {
	Count      int
	Failure    FlatFailure
	Finished   time.Time
	Network    string
	Operation  string
	RemoteAddr string
	Started    time.Time
}

// FlatQUICTLSHandshakeEvent contains a QUIC or TLS handshake event.
type FlatQUICTLSHandshakeEvent struct {
	ALPN            []string
	CipherSuite     string
	Failure         FlatFailure
	Finished        time.Time
	NegotiatedProto string
	Network         string
	PeerCerts       [][]byte
	RemoteAddr      string
	SNI             string
	SkipVerify      bool
	Started         time.Time
	TLSVersion      string
}
