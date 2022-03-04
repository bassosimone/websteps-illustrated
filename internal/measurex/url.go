package measurex

import (
	"log"
	"net"
	"net/http"
	"net/url"
)

// URLMeasurement is the result of measuring an URL.
type URLMeasurement struct {
	// ID is the unique ID of this URLMeasurement.
	ID int64

	// URL is the underlying URL to measure.
	URL *url.URL

	// Cookies contains the list of cookies to use.
	Cookies []*http.Cookie

	// Headers contains request headers.
	Headers http.Header

	// SNI contains the SNI.
	SNI string

	// ALPN contains values for ALPN.
	ALPN []string

	// Host is the host header.
	Host string

	// DNS contains a list of DNS measurements.
	DNS []*DNSLookupMeasurement

	// Endpoint contains a list of endpoint measurements.
	Endpoint []*EndpointMeasurement
}

// Domain is the domain inside the input URL.
func (um *URLMeasurement) Domain() string {
	return um.URL.Hostname()
}

// NewURLMeasurement creates a new URLMeasurement from a string URL.
func (mx *Measurer) NewURLMeasurement(input string) (*URLMeasurement, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, ErrUnknownURLScheme
	}
	out := &URLMeasurement{
		ID:       mx.NextID(),
		URL:      parsed,
		Cookies:  []*http.Cookie{},
		Headers:  map[string][]string{},
		SNI:      parsed.Hostname(),
		ALPN:     []string{},
		Host:     parsed.Hostname(),
		DNS:      []*DNSLookupMeasurement{},
		Endpoint: []*EndpointMeasurement{},
	}
	return out, nil
}

// NewDNSLookupPlan creates a NewDNSLookupPlan for this URLMeasurement. The plan calls
// for resolving the domain name inside um.URL using all the given resolvers.
func (um *URLMeasurement) NewDNSLookupPlan(ri []*DNSResolverInfo) *DNSLookupPlan {
	return &DNSLookupPlan{
		URLMeasurementID: um.ID,
		URL:              um.URL,
		Resolvers:        ri,
	}
}

// URLAddress is an address associated with a given URL.
type URLAddress struct {
	// URLMeasurementID is the ID of the parent URLMeasurement.
	URLMeasurementID int64

	// URL is the original URL.
	URL *url.URL

	// Address is the target IPv4/IPv6 address.
	Address string

	// Flags contains feature flags.
	Flags int64
}

const (
	// urlAddressFlagHTTP3 indicates that a given URL address supports HTTP3.
	urlAddressFlagHTTP3 = 1 << iota

	// urlAddressAlreadyTestedHTTP indicates that this address has already
	// been tested using the cleartext HTTP protocol.
	urlAddressAlreadyTestedHTTP

	// urlAddressAlreadyTestedHTTPS indicates that this address has already
	// been tested using the encrypted HTTPS protocol.
	urlAddressAlreadyTestedHTTPS

	// urlAddressAlreadyTestedHTTP3 indicates that this address has already
	// been tested using the encrypted HTTP3 protocol.
	urlAddressAlreadyTestedHTTP3
)

// Domain returns the domain for which the address should be valid. Because the
// DNS may be lying to us, we cannot be sure about that, though.
func (ua *URLAddress) Domain() string {
	return ua.URL.Hostname()
}

// SupportsHTTP3 returns whether we think this address supports HTTP3.
func (ua *URLAddress) SupportsHTTP3() bool {
	return (ua.Flags & urlAddressFlagHTTP3) != 0
}

// AlreadyTestedHTTP returns whether we've already tested this IP address using HTTP.
func (ua *URLAddress) AlreadyTestedHTTP() bool {
	return (ua.Flags & urlAddressAlreadyTestedHTTP) != 0
}

// AlreadyTestedHTTPS returns whether we've already tested this IP address using HTTPS.
func (ua *URLAddress) AlreadyTestedHTTPS() bool {
	return (ua.Flags & urlAddressAlreadyTestedHTTPS) != 0
}

// AlreadyTestedHTTP3 returns whether we've already tested this IP address using HTTP3.
func (ua *URLAddress) AlreadyTestedHTTP3() bool {
	return (ua.Flags & urlAddressAlreadyTestedHTTP3) != 0
}

// URLAddressList generates a list of URLAddresses based on DNS lookups. The boolean
// return value indicates whether we have at least one IP address in the result.
func (um *URLMeasurement) URLAddressList() ([]*URLAddress, bool) {
	uniq := make(map[string]int64)
	// start searching into the DNS lookup results.
	for _, dns := range um.DNS {
		var flags int64
		if dns.SupportsHTTP3() {
			flags |= urlAddressFlagHTTP3
		}
		for _, addr := range dns.Addresses {
			if net.ParseIP(addr) == nil {
				// Skip CNAMEs in case they slip through.
				log.Printf("cannot parse %+v inside um.DNS as IP address", addr)
				continue
			}
			uniq[addr] |= flags
		}
	}
	// continue searching into HTTP responses.
	for _, epnt := range um.Endpoint {
		ipAddr, err := epnt.IPAddress()
		if err != nil {
			// This may actually be an IPv6 address with explicit scope
			log.Printf("cannot parse %+v inside epnt.Address as IP address", epnt.Address)
			continue
		}
		if epnt.IsHTTPMeasurement() {
			uniq[ipAddr] |= urlAddressAlreadyTestedHTTP
		}
		if epnt.IsHTTPSMeasurement() {
			uniq[ipAddr] |= urlAddressAlreadyTestedHTTPS
		}
		if epnt.IsHTTP3Measurement() {
			uniq[ipAddr] |= urlAddressAlreadyTestedHTTP3
		}
		if !epnt.SupportsAltSvcHTTP3() {
			continue
		}
		uniq[ipAddr] |= urlAddressFlagHTTP3
	}
	// finally build the return list.
	out := make([]*URLAddress, 8)
	for addr, flags := range uniq {
		out = append(out, &URLAddress{
			URLMeasurementID: um.ID,
			URL:              um.URL,
			Address:          addr,
			Flags:            flags,
		})
	}
	return out, len(out) > 0
}

// NewEndpointMeasurementPlanForHTTP creates a new plan for measuring all the
// endpoints that have not been measured yet using HTTP.
func (um *URLMeasurement) NewEndpointMeasurementPlanForHTTP() ([]*EndpointMeasurementPlan, bool) {
	addrs, _ := um.URLAddressList()
	out := make([]*EndpointMeasurementPlan, 8)
	for _, addr := range addrs {
		if addr.AlreadyTestedHTTP() {
			continue
		}
		epnt, err := um.makeEndpoint(addr.Address)
		if err != nil {
			log.Printf("cannot make endpoint: %s", err.Error())
			continue
		}
		out = append(out, &EndpointMeasurementPlan{
			URLMeasurementID: um.ID,
			Domain:           um.Domain(),
			Network:          NetworkTCP,
			Address:          epnt,
			SNI:              "",         // not needed
			ALPN:             []string{}, // not needed
			URL:              um.URL,
			Header:           NewHTTPRequestHeaderForMeasuring(),
			Cookies:          um.Cookies,
		})
	}
	return out, len(out) > 0
}

// NewEndpointMeasurementPlanForHTTPS creates a new plan for measuring all the
// endpoints that have not been measured yet using HTTPS.
func (um *URLMeasurement) NewEndpointMeasurementPlanForHTTPS() ([]*EndpointMeasurementPlan, bool) {
	addrs, _ := um.URLAddressList()
	out := make([]*EndpointMeasurementPlan, 8)
	for _, addr := range addrs {
		if addr.AlreadyTestedHTTPS() {
			continue
		}
		epnt, err := um.makeEndpoint(addr.Address)
		if err != nil {
			log.Printf("cannot make endpoint: %s", err.Error())
			continue
		}
		out = append(out, &EndpointMeasurementPlan{
			URLMeasurementID: um.ID,
			Domain:           um.Domain(),
			Network:          NetworkTCP,
			Address:          epnt,
			SNI:              um.Domain(),
			ALPN:             []string{"h2", "http/1.1"},
			URL:              um.URL,
			Header:           NewHTTPRequestHeaderForMeasuring(),
			Cookies:          um.Cookies,
		})
	}
	return out, len(out) > 0
}

// NewEndpointMeasurementPlanForHTTP3 creates a new plan for measuring all the
// endpoints that have not been measured yet using HTTP3.
func (um *URLMeasurement) NewEndpointMeasurementPlanForHTTP3() ([]*EndpointMeasurementPlan, bool) {
	addrs, _ := um.URLAddressList()
	out := make([]*EndpointMeasurementPlan, 8)
	for _, addr := range addrs {
		if addr.AlreadyTestedHTTP3() {
			continue
		}
		epnt, err := um.makeEndpoint(addr.Address)
		if err != nil {
			log.Printf("cannot make endpoint: %s", err.Error())
			continue
		}
		out = append(out, &EndpointMeasurementPlan{
			URLMeasurementID: um.ID,
			Domain:           um.Domain(),
			Network:          NetworkQUIC,
			Address:          epnt,
			SNI:              um.Domain(),
			ALPN:             []string{"h3"},
			URL:              um.URL,
			Header:           NewHTTPRequestHeaderForMeasuring(),
			Cookies:          um.Cookies,
		})
	}
	return out, len(out) > 0
}

// makeEndpoint makes a level-4 endpoint given the address and either the URL scheme
// or the explicit port provided inside the URL.
func (um *URLMeasurement) makeEndpoint(address string) (string, error) {
	port, err := PortFromURL(um.URL)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(address, port), nil
}
