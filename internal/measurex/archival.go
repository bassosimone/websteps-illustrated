package measurex

import (
	"time"

	"github.com/bassosimone/websteps-illustrated/internal/archival"
	"github.com/bassosimone/websteps-illustrated/internal/model"
)

//
// Archival
//
// Convert to the archival data format.
//

// ArchivalDNSLookupMeasurement is an archival DNS lookup measurement.
type ArchivalDNSLookupMeasurement struct {
	// ID is the unique ID of this measurement.
	ID int64 `json:"id"`

	// URLMeasurementID is the ID of the parent URLMeasurement.
	URLMeasurementID int64 `json:"url_measurement_id"`

	// DNSRoundTrips contains DNS round trips.
	DNSRoundTrips []model.ArchivalDNSRoundTripEvent `json:"dns_round_trips"`

	// Queries contains the DNS lookup events.
	Queries []model.ArchivalDNSLookupResult `json:"queries"`
}

// ToArchival converts a DNSLookupMeasurement to ArchivalDNSLookupMeasurement.
func (m *DNSLookupMeasurement) ToArchival(begin time.Time) ArchivalDNSLookupMeasurement {
	return ArchivalDNSLookupMeasurement{
		ID:               m.ID,
		URLMeasurementID: m.URLMeasurementID,
		DNSRoundTrips:    archival.NewArchivalDNSRoundTripEventList(begin, m.RoundTrip),
		Queries:          m.Lookup.ToArchival(begin),
	}
}

// ArchivalEndpointMeasurement is the archival format of an endpoint measurement.
type ArchivalEndpointMeasurement struct {
	// ID is the unique ID of this measurement.
	ID int64 `json:"id"`

	// URLMeasurementID is the ID of the URLMeasurement that created us.
	URLMeasurementID int64 `json:"url_measurement_id"`

	// URL is the URL we're fetching.
	URL string `json:"url"`

	// Endpoint is the endpoint address.
	Endpoint string `json:"endpoint"`

	// Failure is the error that occurred.
	Failure *string `json:"failure"`

	// FailedOperation is the operation that failed.
	FailedOperation *string `json:"failed_operation"`

	// NetworkEvent contains network events (if any).
	NetworkEvents []model.ArchivalNetworkEvent `json:"network_events"`

	// TCPConnect contains the TCP connect event (if any).
	TCPConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// QUICTLSHandshake contains the QUIC/TLS handshake event (if any).
	QUICTLSHandshake *model.ArchivalTLSOrQUICHandshakeResult `json:"quic_tls_handshake"`

	// HTTPRoundTrip contains the HTTP round trip event (if any).
	HTTPRoundTrip *model.ArchivalHTTPRequestResult `json:"request"`
}

// ToArchival converts a EndpointMeasurement to ArchivalEndpointMeasurement.
func (m *EndpointMeasurement) ToArchival(begin time.Time) ArchivalEndpointMeasurement {
	return ArchivalEndpointMeasurement{
		ID:               m.ID,
		URLMeasurementID: m.URLMeasurementID,
		URL:              m.URL.String(),
		Endpoint:         m.EndpointAddress(),
		Failure:          m.Failure.ToArchivalFailure(),
		FailedOperation:  m.FailedOperation.ToArchivalFailure(),
		NetworkEvents:    archival.NewArchivalNetworkEventList(begin, m.NetworkEvent),
		TCPConnect:       m.toArchivalTCPConnectResult(begin),
		QUICTLSHandshake: m.toArchivalTLSOrQUICHandshakeResult(begin),
		HTTPRoundTrip:    m.toArchivalHTTPRequestResult(begin),
	}
}

func (m *EndpointMeasurement) toArchivalTCPConnectResult(begin time.Time) (out *model.ArchivalTCPConnectResult) {
	if m.TCPConnect != nil {
		v := m.TCPConnect.ToArchivalTCPConnectResult(begin)
		out = &v
	}
	return
}

func (m *EndpointMeasurement) toArchivalTLSOrQUICHandshakeResult(
	begin time.Time) (out *model.ArchivalTLSOrQUICHandshakeResult) {
	if m.QUICTLSHandshake != nil {
		v := m.QUICTLSHandshake.ToArchival(begin)
		out = &v
	}
	return
}

func (m *EndpointMeasurement) toArchivalHTTPRequestResult(begin time.Time) (out *model.ArchivalHTTPRequestResult) {
	if m.HTTPRoundTrip != nil {
		v := m.HTTPRoundTrip.ToArchival(begin)
		out = &v
	}
	return
}

// ArchivalURLMeasurement is the archival format of an URL measurement.
type ArchivalURLMeasurement struct {
	// ID is the unique ID of this URLMeasurement.
	ID int64 `json:"id"`

	// EndpointIDs contains the ID of the EndpointMeasurement(s) that
	// generated this URLMeasurement through redirects.
	EndpointIDs []int64 `json:"endpoint_ids"`

	// URL is the underlying URL to measure.
	URL string `json:"url"`

	// SNI contains the SNI.
	SNI string `json:"sni,omitempty"`

	// ALPN contains values for ALPN.
	ALPN []string `json:"alpn,omitempty"`

	// Host is the host header.
	Host string `json:"host_header,omitempty"`

	// DNS contains a list of DNS measurements.
	DNS []ArchivalDNSLookupMeasurement `json:"dns"`

	// Endpoint contains a list of endpoint measurements.
	Endpoint []ArchivalEndpointMeasurement `json:"endpoint"`
}

// ToArchival converts URLMeasurement to ArchivalURLMeasurement.
func (m *URLMeasurement) ToArchival(begin time.Time) ArchivalURLMeasurement {
	return ArchivalURLMeasurement{
		ID:          m.ID,
		EndpointIDs: m.EndpointIDs,
		URL:         m.URL.String(),
		SNI:         m.SNI,
		ALPN:        m.ALPN,
		Host:        m.Host,
		DNS:         NewArchivalDNSLookupMeasurementList(begin, m.DNS),
		Endpoint:    NewArchivalEndpointMeasurementList(begin, m.Endpoint),
	}
}

// NewArchivalDNSLookupMeasurementList converts a []*DNSLookupMeasurement into
// a []ArchivalDNSLookupMeasurement.
func NewArchivalDNSLookupMeasurementList(
	begin time.Time, in []*DNSLookupMeasurement) (out []ArchivalDNSLookupMeasurement) {
	for _, entry := range in {
		out = append(out, entry.ToArchival(begin))
	}
	return
}

// NewArchivalEndpointMeasurementList converts a []*EndpointMeasurement into
// a []ArchivalEndpointMeasurement.
func NewArchivalEndpointMeasurementList(
	begin time.Time, in []*EndpointMeasurement) (out []ArchivalEndpointMeasurement) {
	for _, entry := range in {
		out = append(out, entry.ToArchival(begin))
	}
	return
}