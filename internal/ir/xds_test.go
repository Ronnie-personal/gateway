// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package ir

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

var (
	// HTTPListener
	happyHTTPListener = HTTPListener{
		Name:      "happy",
		Address:   "0.0.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		Routes:    []*HTTPRoute{&happyHTTPRoute},
	}
	happyHTTPSListener = HTTPListener{
		Name:      "happy",
		Address:   "0.0.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		TLS: []*TLSListenerConfig{{
			Name:              "happy",
			ServerCertificate: []byte{1, 2, 3},
			PrivateKey:        []byte{1, 2, 3},
		}},
		Routes: []*HTTPRoute{&happyHTTPRoute},
	}
	invalidAddrHTTPListener = HTTPListener{
		Name:      "invalid-addr",
		Address:   "1.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		Routes:    []*HTTPRoute{&happyHTTPRoute},
	}
	invalidRouteMatchHTTPListener = HTTPListener{
		Name:      "invalid-route-match",
		Address:   "0.0.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		Routes:    []*HTTPRoute{&emptyMatchHTTPRoute},
	}
	invalidBackendHTTPListener = HTTPListener{
		Name:      "invalid-backend-match",
		Address:   "0.0.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		Routes:    []*HTTPRoute{&invalidBackendHTTPRoute},
	}
	weightedInvalidBackendsHTTPListener = HTTPListener{
		Name:      "weighted-invalid-backends-match",
		Address:   "0.0.0.0",
		Port:      80,
		Hostnames: []string{"example.com"},
		Routes:    []*HTTPRoute{&weightedInvalidBackendsHTTPRoute},
	}

	// TCPListener
	happyTCPListenerTLSPassthrough = TCPListener{
		Name:        "happy",
		Address:     "0.0.0.0",
		Port:        80,
		TLS:         &TLS{Passthrough: &TLSInspectorConfig{SNIs: []string{"example.com"}}},
		Destination: &happyRouteDestination,
	}

	happyTCPListenerTLSTerminate = TCPListener{
		Name:    "happy",
		Address: "0.0.0.0",
		Port:    80,
		TLS: &TLS{Terminate: []*TLSListenerConfig{{
			Name:              "happy",
			ServerCertificate: []byte("server-cert"),
			PrivateKey:        []byte("priv-key"),
		}}},
		Destination: &happyRouteDestination,
	}

	emptySNITCPListenerTLSPassthrough = TCPListener{
		Name:        "empty-sni",
		Address:     "0.0.0.0",
		Port:        80,
		Destination: &happyRouteDestination,
	}
	invalidNameTCPListenerTLSPassthrough = TCPListener{
		Address:     "0.0.0.0",
		Port:        80,
		TLS:         &TLS{Passthrough: &TLSInspectorConfig{SNIs: []string{"example.com"}}},
		Destination: &happyRouteDestination,
	}
	invalidAddrTCPListenerTLSPassthrough = TCPListener{
		Name:        "invalid-addr",
		Address:     "1.0.0",
		Port:        80,
		TLS:         &TLS{Passthrough: &TLSInspectorConfig{SNIs: []string{"example.com"}}},
		Destination: &happyRouteDestination,
	}
	invalidSNITCPListenerTLSPassthrough = TCPListener{
		Address:     "0.0.0.0",
		Port:        80,
		TLS:         &TLS{Passthrough: &TLSInspectorConfig{SNIs: []string{}}},
		Destination: &happyRouteDestination,
	}

	// UDPListener
	happyUDPListener = UDPListener{
		Name:        "happy",
		Address:     "0.0.0.0",
		Port:        80,
		Destination: &happyRouteDestination,
	}
	invalidNameUDPListener = UDPListener{
		Address:     "0.0.0.0",
		Port:        80,
		Destination: &happyRouteDestination,
	}
	invalidAddrUDPListener = UDPListener{
		Name:        "invalid-addr",
		Address:     "1.0.0",
		Port:        80,
		Destination: &happyRouteDestination,
	}
	invalidPortUDPListenerT = UDPListener{
		Name:        "invalid-port",
		Address:     "0.0.0.0",
		Port:        0,
		Destination: &happyRouteDestination,
	}

	// HTTPRoute
	happyHTTPRoute = HTTPRoute{
		Name:     "happy",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("example"),
		},
		Destination: &happyRouteDestination,
	}
	emptyMatchHTTPRoute = HTTPRoute{
		Name:        "empty-match",
		Hostname:    "*",
		Destination: &happyRouteDestination,
	}
	invalidBackendHTTPRoute = HTTPRoute{
		Name:     "invalid-backend",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("invalid-backend"),
		},
		BackendWeights: BackendWeights{
			Invalid: 1,
		},
	}
	weightedInvalidBackendsHTTPRoute = HTTPRoute{
		Name:     "weighted-invalid-backends",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("invalid-backends"),
		},
		Destination: &happyRouteDestination,
		BackendWeights: BackendWeights{
			Invalid: 1,
			Valid:   1,
		},
	}

	redirectHTTPRoute = HTTPRoute{
		Name:     "redirect",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("redirect"),
		},
		Redirect: &Redirect{
			Scheme:   ptrTo("https"),
			Hostname: ptrTo("redirect.example.com"),
			Path: &HTTPPathModifier{
				FullReplace: ptrTo("/redirect"),
			},
			Port:       ptrTo(uint32(8443)),
			StatusCode: ptrTo(int32(301)),
		},
	}
	// A direct response error is used when an invalid filter type is supplied
	invalidFilterHTTPRoute = HTTPRoute{
		Name:     "filter-error",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("filter-error"),
		},
		DirectResponse: &DirectResponse{
			Body:       ptrTo("invalid filter type"),
			StatusCode: uint32(500),
		},
	}

	redirectFilterInvalidStatus = HTTPRoute{
		Name:     "redirect-bad-status-scheme-nopat",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("redirect"),
		},
		Redirect: &Redirect{
			Scheme:     ptrTo("err"),
			Hostname:   ptrTo("redirect.example.com"),
			Path:       &HTTPPathModifier{},
			Port:       ptrTo(uint32(8443)),
			StatusCode: ptrTo(int32(305)),
		},
	}
	redirectFilterBadPath = HTTPRoute{
		Name:     "redirect",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("redirect"),
		},
		Redirect: &Redirect{
			Scheme:   ptrTo("https"),
			Hostname: ptrTo("redirect.example.com"),
			Path: &HTTPPathModifier{
				FullReplace:        ptrTo("/redirect"),
				PrefixMatchReplace: ptrTo("/redirect"),
			},
			Port:       ptrTo(uint32(8443)),
			StatusCode: ptrTo(int32(301)),
		},
	}
	directResponseBadStatus = HTTPRoute{
		Name:     "redirect",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("redirect"),
		},
		DirectResponse: &DirectResponse{
			Body:       ptrTo("invalid filter type"),
			StatusCode: uint32(799),
		},
	}

	urlRewriteHTTPRoute = HTTPRoute{
		Name:     "rewrite",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("rewrite"),
		},
		URLRewrite: &URLRewrite{
			Hostname: ptrTo("rewrite.example.com"),
			Path: &HTTPPathModifier{
				FullReplace: ptrTo("/rewrite"),
			},
		},
	}

	urlRewriteFilterBadPath = HTTPRoute{
		Name:     "rewrite",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("rewrite"),
		},
		URLRewrite: &URLRewrite{
			Hostname: ptrTo("rewrite.example.com"),
			Path: &HTTPPathModifier{
				FullReplace:        ptrTo("/rewrite"),
				PrefixMatchReplace: ptrTo("/rewrite"),
			},
		},
	}

	addRequestHeaderHTTPRoute = HTTPRoute{
		Name:     "addheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("addheader"),
		},
		AddRequestHeaders: []AddHeader{
			{
				Name:   "example-header",
				Value:  "example-value",
				Append: true,
			},
			{
				Name:   "example-header-2",
				Value:  "example-value-2",
				Append: false,
			},
			{
				Name:   "empty-header",
				Value:  "",
				Append: false,
			},
		},
	}

	removeRequestHeaderHTTPRoute = HTTPRoute{
		Name:     "remheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("remheader"),
		},
		RemoveRequestHeaders: []string{
			"x-request-header",
			"example-header",
			"another-header",
		},
	}

	addAndRemoveRequestHeadersDupeHTTPRoute = HTTPRoute{
		Name:     "duplicateheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("duplicateheader"),
		},
		AddRequestHeaders: []AddHeader{
			{
				Name:   "example-header",
				Value:  "example-value",
				Append: true,
			},
			{
				Name:   "example-header",
				Value:  "example-value-2",
				Append: false,
			},
		},
		RemoveRequestHeaders: []string{
			"x-request-header",
			"example-header",
			"example-header",
		},
	}

	addRequestHeaderEmptyHTTPRoute = HTTPRoute{
		Name:     "addemptyheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("addemptyheader"),
		},
		AddRequestHeaders: []AddHeader{
			{
				Name:   "",
				Value:  "example-value",
				Append: true,
			},
		},
	}

	addResponseHeaderHTTPRoute = HTTPRoute{
		Name:     "addheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("addheader"),
		},
		AddResponseHeaders: []AddHeader{
			{
				Name:   "example-header",
				Value:  "example-value",
				Append: true,
			},
			{
				Name:   "example-header-2",
				Value:  "example-value-2",
				Append: false,
			},
			{
				Name:   "empty-header",
				Value:  "",
				Append: false,
			},
		},
	}

	removeResponseHeaderHTTPRoute = HTTPRoute{
		Name:     "remheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("remheader"),
		},
		RemoveResponseHeaders: []string{
			"x-request-header",
			"example-header",
			"another-header",
		},
	}

	addAndRemoveResponseHeadersDupeHTTPRoute = HTTPRoute{
		Name:     "duplicateheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("duplicateheader"),
		},
		AddResponseHeaders: []AddHeader{
			{
				Name:   "example-header",
				Value:  "example-value",
				Append: true,
			},
			{
				Name:   "example-header",
				Value:  "example-value-2",
				Append: false,
			},
		},
		RemoveResponseHeaders: []string{
			"x-request-header",
			"example-header",
			"example-header",
		},
	}

	addResponseHeaderEmptyHTTPRoute = HTTPRoute{
		Name:     "addemptyheader",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("addemptyheader"),
		},
		AddResponseHeaders: []AddHeader{
			{
				Name:   "",
				Value:  "example-value",
				Append: true,
			},
		},
	}

	jwtAuthenHTTPRoute = HTTPRoute{
		Name:     "jwtauthen",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("jwtauthen"),
		},
		RequestAuthentication: &RequestAuthentication{
			JWT: &JwtRequestAuthentication{
				Providers: []egv1a1.JwtAuthenticationFilterProvider{
					{
						Name: "test1",
						RemoteJWKS: egv1a1.RemoteJWKS{
							URI: "https://test1.local",
						},
					},
				},
			},
		},
	}
	requestMirrorFilter = HTTPRoute{
		Name:     "mirrorfilter",
		Hostname: "*",
		PathMatch: &StringMatch{
			Exact: ptrTo("mirrorfilter"),
		},
		Mirror: []*RouteDestination{&happyRouteDestination},
	}

	// RouteDestination
	happyRouteDestination = RouteDestination{
		Name: "happy-dest",
		Endpoints: []*DestinationEndpoint{
			{
				Host: "10.11.12.13",
				Port: 8080,
			},
		},
	}
)

// Creates a pointer to any type
func ptrTo[T any](x T) *T {
	return &x
}

func TestValidateXds(t *testing.T) {
	tests := []struct {
		name  string
		input Xds
		want  []error
	}{
		{
			name: "happy",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPListener},
			},
			want: nil,
		},
		{
			name: "happy tls passthrough",
			input: Xds{
				TCP: []*TCPListener{&happyTCPListenerTLSPassthrough},
			},
			want: nil,
		},
		{
			name: "happy tls terminate",
			input: Xds{
				TCP: []*TCPListener{&happyTCPListenerTLSTerminate},
			},
			want: nil,
		},
		{
			name: "invalid listener",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPListener, &invalidAddrHTTPListener, &invalidRouteMatchHTTPListener},
			},
			want: []error{ErrListenerAddressInvalid, ErrHTTPRouteMatchEmpty},
		},
		{
			name: "invalid backend",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPListener, &invalidBackendHTTPListener},
			},
			want: nil,
		},
		{
			name: "weighted invalid backend",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPListener, &weightedInvalidBackendsHTTPListener},
			},
			want: nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				got := test.input.Validate()
				for _, w := range test.want {
					assert.ErrorContains(t, got, w.Error())
				}
			}
		})
	}
}

func TestValidateHTTPListener(t *testing.T) {
	tests := []struct {
		name  string
		input HTTPListener
		want  []error
	}{
		{
			name:  "happy",
			input: happyHTTPListener,
			want:  nil,
		},
		{
			name: "invalid name",
			input: HTTPListener{
				Address:   "0.0.0.0",
				Port:      80,
				Hostnames: []string{"example.com"},
				Routes:    []*HTTPRoute{&happyHTTPRoute},
			},
			want: []error{ErrListenerNameEmpty},
		},
		{
			name:  "invalid addr",
			input: invalidAddrHTTPListener,
			want:  []error{ErrListenerAddressInvalid},
		},
		{
			name: "invalid port and hostnames",
			input: HTTPListener{
				Name:    "invalid-port-and-hostnames",
				Address: "1.0.0",
				Routes:  []*HTTPRoute{&happyHTTPRoute},
			},
			want: []error{ErrListenerPortInvalid, ErrHTTPListenerHostnamesEmpty},
		},
		{
			name:  "invalid route match",
			input: invalidRouteMatchHTTPListener,
			want:  []error{ErrHTTPRouteMatchEmpty},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				got := test.input.Validate()
				for _, w := range test.want {
					assert.ErrorContains(t, got, w.Error())
				}
			}
		})
	}
}

func TestValidateTCPListener(t *testing.T) {
	tests := []struct {
		name  string
		input TCPListener
		want  []error
	}{
		{
			name:  "tls passthrough happy",
			input: happyTCPListenerTLSPassthrough,
			want:  nil,
		},
		{
			name:  "tcp empty SNIs",
			input: emptySNITCPListenerTLSPassthrough,
			want:  nil,
		},
		{
			name:  "tls passthrough invalid name",
			input: invalidNameTCPListenerTLSPassthrough,
			want:  []error{ErrListenerNameEmpty},
		},
		{
			name:  "tls passthrough invalid addr",
			input: invalidAddrTCPListenerTLSPassthrough,
			want:  []error{ErrListenerAddressInvalid},
		},
		{
			name:  "tls passthrough empty SNIs",
			input: invalidSNITCPListenerTLSPassthrough,
			want:  []error{ErrTCPListenerSNIsEmpty},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				got := test.input.Validate()
				for _, w := range test.want {
					assert.ErrorContains(t, got, w.Error())
				}
			}
		})
	}
}

func TestValidateTLSListenerConfig(t *testing.T) {
	tests := []struct {
		name  string
		input TLSListenerConfig
		want  error
	}{
		{
			name: "happy",
			input: TLSListenerConfig{
				ServerCertificate: []byte("server-cert"),
				PrivateKey:        []byte("priv-key"),
			},
			want: nil,
		},
		{
			name: "invalid server cert",
			input: TLSListenerConfig{
				PrivateKey: []byte("priv-key"),
			},
			want: ErrTLSServerCertEmpty,
		},
		{
			name: "invalid private key",
			input: TLSListenerConfig{
				ServerCertificate: []byte("server-cert"),
			},
			want: ErrTLSPrivateKey,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				require.EqualError(t, test.input.Validate(), test.want.Error())
			}
		})
	}
}

func TestEqualXds(t *testing.T) {
	tests := []struct {
		desc  string
		a     *Xds
		b     *Xds
		equal bool
	}{
		{
			desc: "out of order tcp listeners are equal",
			a: &Xds{
				TCP: []*TCPListener{
					{Name: "listener-1"},
					{Name: "listener-2"},
				},
			},
			b: &Xds{
				TCP: []*TCPListener{
					{Name: "listener-2"},
					{Name: "listener-1"},
				},
			},
			equal: true,
		},
		{
			desc: "out of order http routes are equal",
			a: &Xds{
				HTTP: []*HTTPListener{
					{
						Name: "listener-1",
						Routes: []*HTTPRoute{
							{Name: "route-1"},
							{Name: "route-2"},
						},
					},
				},
			},
			b: &Xds{
				HTTP: []*HTTPListener{
					{
						Name: "listener-1",
						Routes: []*HTTPRoute{
							{Name: "route-2"},
							{Name: "route-1"},
						},
					},
				},
			},
			equal: true,
		},
		{
			desc: "out of order udp listeners are equal",
			a: &Xds{
				UDP: []*UDPListener{
					{Name: "listener-1"},
					{Name: "listener-2"},
				},
			},
			b: &Xds{
				UDP: []*UDPListener{
					{Name: "listener-2"},
					{Name: "listener-1"},
				},
			},
			equal: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.equal, cmp.Equal(tc.a, tc.b))
		})
	}
}

func TestValidateUDPListener(t *testing.T) {
	tests := []struct {
		name  string
		input UDPListener
		want  []error
	}{
		{
			name:  "udp happy",
			input: happyUDPListener,
			want:  nil,
		},
		{
			name:  "udp invalid name",
			input: invalidNameUDPListener,
			want:  []error{ErrListenerNameEmpty},
		},
		{
			name:  "udp invalid addr",
			input: invalidAddrUDPListener,
			want:  []error{ErrListenerAddressInvalid},
		},
		{
			name:  "udp invalid port",
			input: invalidPortUDPListenerT,
			want:  []error{ErrListenerPortInvalid},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				got := test.input.Validate()
				for _, w := range test.want {
					assert.ErrorContains(t, got, w.Error())
				}
			}
		})
	}
}

func TestValidateHTTPRoute(t *testing.T) {
	tests := []struct {
		name  string
		input HTTPRoute
		want  []error
	}{
		{
			name:  "happy",
			input: happyHTTPRoute,
			want:  nil,
		},
		{
			name: "invalid name",
			input: HTTPRoute{
				Hostname: "*",
				PathMatch: &StringMatch{
					Exact: ptrTo("example"),
				},
				Destination: &happyRouteDestination,
			},
			want: []error{ErrHTTPRouteNameEmpty},
		},
		{
			name: "invalid hostname",
			input: HTTPRoute{
				Name: "invalid hostname",
				PathMatch: &StringMatch{
					Exact: ptrTo("example"),
				},
				Destination: &happyRouteDestination,
			},
			want: []error{ErrHTTPRouteHostnameEmpty},
		},
		{
			name:  "empty match",
			input: emptyMatchHTTPRoute,
			want:  []error{ErrHTTPRouteMatchEmpty},
		},
		{
			name:  "invalid backend",
			input: invalidBackendHTTPRoute,
			want:  nil,
		},
		{
			name:  "weighted invalid backends",
			input: weightedInvalidBackendsHTTPRoute,
			want:  nil,
		},
		{
			name: "empty name and invalid match",
			input: HTTPRoute{
				Hostname:      "*",
				HeaderMatches: []*StringMatch{ptrTo(StringMatch{})},
				Destination:   &happyRouteDestination,
			},
			want: []error{ErrHTTPRouteNameEmpty, ErrStringMatchConditionInvalid},
		},
		{
			name:  "redirect-httproute",
			input: redirectHTTPRoute,
			want:  nil,
		},
		{
			name:  "filter-error-httproute",
			input: invalidFilterHTTPRoute,
			want:  nil,
		},
		{
			name:  "redirect-bad-status-scheme-nopath",
			input: redirectFilterInvalidStatus,
			want:  []error{ErrRedirectUnsupportedStatus, ErrRedirectUnsupportedScheme, ErrHTTPPathModifierNoReplace},
		},
		{
			name:  "redirect-bad-path",
			input: redirectFilterBadPath,
			want:  []error{ErrHTTPPathModifierDoubleReplace},
		},
		{
			name:  "direct-response-bad-status",
			input: directResponseBadStatus,
			want:  []error{ErrDirectResponseStatusInvalid},
		},
		{
			name:  "rewrite-httproute",
			input: urlRewriteHTTPRoute,
			want:  nil,
		},
		{
			name:  "rewrite-bad-path",
			input: urlRewriteFilterBadPath,
			want:  []error{ErrHTTPPathModifierDoubleReplace},
		},
		{
			name:  "add-request-headers-httproute",
			input: addRequestHeaderHTTPRoute,
			want:  nil,
		},
		{
			name:  "remove-request-headers-httproute",
			input: removeRequestHeaderHTTPRoute,
			want:  nil,
		},
		{
			name:  "add-remove-request-headers-duplicate",
			input: addAndRemoveRequestHeadersDupeHTTPRoute,
			want:  []error{ErrAddHeaderDuplicate, ErrRemoveHeaderDuplicate},
		},
		{
			name:  "add-request-header-empty",
			input: addRequestHeaderEmptyHTTPRoute,
			want:  []error{ErrAddHeaderEmptyName},
		},
		{
			name:  "add-response-headers-httproute",
			input: addResponseHeaderHTTPRoute,
			want:  nil,
		},
		{
			name:  "remove-response-headers-httproute",
			input: removeResponseHeaderHTTPRoute,
			want:  nil,
		},
		{
			name:  "add-remove-response-headers-duplicate",
			input: addAndRemoveResponseHeadersDupeHTTPRoute,
			want:  []error{ErrAddHeaderDuplicate, ErrRemoveHeaderDuplicate},
		},
		{
			name:  "add-response-header-empty",
			input: addResponseHeaderEmptyHTTPRoute,
			want:  []error{ErrAddHeaderEmptyName},
		},
		{
			name:  "jwt-authen-httproute",
			input: jwtAuthenHTTPRoute,
		},
		{
			name:  "mirror-filter",
			input: requestMirrorFilter,
			want:  nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				got := test.input.Validate()
				for _, w := range test.want {
					assert.ErrorContains(t, got, w.Error())
				}
			}
		})
	}
}

func TestValidateRouteDestination(t *testing.T) {
	tests := []struct {
		name  string
		input RouteDestination
		want  error
	}{
		{
			name:  "happy",
			input: happyRouteDestination,
			want:  nil,
		},
		{
			name: "invalid ip",
			input: RouteDestination{
				Name: "invalid ip",
				Endpoints: []*DestinationEndpoint{
					{
						Host: "example.com",
						Port: 8080,
					},
				},
			},
			want: ErrDestEndpointHostInvalid,
		},
		{
			name: "missing ip",
			input: RouteDestination{
				Name: "missing ip",
				Endpoints: []*DestinationEndpoint{
					{
						Port: 8080,
					},
				},
			},
			want: ErrDestEndpointHostInvalid,
		},
		{
			name: "missing port",
			input: RouteDestination{
				Name: "missing port",
				Endpoints: []*DestinationEndpoint{
					{
						Host: "10.11.12.13",
					},
				},
			},
			want: ErrDestEndpointPortInvalid,
		},
		{
			name: "missing name",
			input: RouteDestination{
				Endpoints: []*DestinationEndpoint{
					{
						Host: "10.11.12.13",
						Port: 8080,
					},
				},
			},
			want: ErrDestinationNameEmpty,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				require.EqualError(t, test.input.Validate(), test.want.Error())
			}
		})
	}
}

func TestValidateStringMatch(t *testing.T) {
	tests := []struct {
		name  string
		input StringMatch
		want  error
	}{
		{
			name: "happy",
			input: StringMatch{
				Exact: ptrTo("example"),
			},
			want: nil,
		},
		{
			name:  "no fields set",
			input: StringMatch{},
			want:  ErrStringMatchConditionInvalid,
		},
		{
			name: "multiple fields set",
			input: StringMatch{
				Exact:  ptrTo("example"),
				Name:   "example",
				Prefix: ptrTo("example"),
			},
			want: ErrStringMatchConditionInvalid,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				require.EqualError(t, test.input.Validate(), test.want.Error())
			}
		})
	}
}

func TestValidateJwtRequestAuthentication(t *testing.T) {
	tests := []struct {
		name  string
		input JwtRequestAuthentication
		want  error
	}{
		{
			name: "nil rules",
			input: JwtRequestAuthentication{
				Providers: nil,
			},
			want: nil,
		},
		{
			name: "provider with remote jwks uri",
			input: JwtRequestAuthentication{
				Providers: []egv1a1.JwtAuthenticationFilterProvider{
					{
						Name:      "test",
						Issuer:    "https://test.local",
						Audiences: []string{"test1", "test2"},
						RemoteJWKS: egv1a1.RemoteJWKS{
							URI: "https://test.local",
						},
					},
				},
			},
			want: nil,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			if test.want == nil {
				require.NoError(t, test.input.Validate())
			} else {
				require.EqualError(t, test.input.Validate(), test.want.Error())
			}
		})
	}
}

func TestPrintable(t *testing.T) {
	tests := []struct {
		name  string
		input Xds
		want  *Xds
	}{
		{
			name:  "empty",
			input: Xds{},
			want:  &Xds{},
		},
		{
			name: "http",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPListener},
			},
			want: &Xds{
				HTTP: []*HTTPListener{&happyHTTPListener},
			},
		},
		{
			name: "https",
			input: Xds{
				HTTP: []*HTTPListener{&happyHTTPSListener},
			},
			want: &Xds{
				HTTP: []*HTTPListener{&happyHTTPListener},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, *test.want, *test.input.Printable())
		})
	}
}
