// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"net/url"
	"testing"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/udm/context"
	"github.com/omec-project/udm/factory"
	"github.com/omec-project/util/sbiaddr"
)

const (
	testPort    = 29503
	testIPv4    = "10.0.0.5"
	testIPv6    = "fd00::5"
	testFQDN    = "udm"
	testV4URI   = "https://10.0.0.5:29503"
	testV6URI   = "https://[fd00::5]:29503"
	testFQDNURI = "https://udm:29503"
	testDefault = "https://" + factory.UDM_DEFAULT_IPV4 + ":29503"
	testPodIPv6 = "2001:db8:600::5"
	testAnyV4   = "0.0.0.0"
)

// endPoint is the part of an IpEndPoint the tests compare.
type endPoint struct {
	ipv4, ipv6 string
	port       int32
}

func endPoints(svc models.NFService) []endPoint {
	var out []endPoint
	for _, ep := range svc.GetIpEndPoints() {
		out = append(out, endPoint{ipv4: ep.GetIpv4Address(), ipv6: ep.GetIpv6Address(), port: ep.GetPort()})
	}
	return out
}

func initWithSbi(t *testing.T, sbi *factory.Sbi) *context.UDMContext {
	t.Helper()
	orig := factory.UdmConfig
	t.Cleanup(func() { factory.UdmConfig = orig })
	factory.UdmConfig = factory.Config{
		Info: &factory.Info{Version: "1.0.0"},
		Configuration: &factory.Configuration{
			Sbi:         sbi,
			ServiceList: []string{"nudm-sdm", "nudm-uecm"},
			Keys:        &factory.Keys{},
		},
	}
	ctx := &context.UDMContext{NfService: make(map[models.ServiceName]models.NFService)}
	InitUDMContext(ctx)
	return ctx
}

func TestInitUDMContextRegisterAddrs(t *testing.T) {
	tests := []struct {
		name           string
		env            map[string]string
		sbi            factory.Sbi
		wantIPv4       string // context.RegisterIPv4
		wantIPv6       string // context.RegisterIPv6
		wantURI        string
		wantFqdn       string // NFService.fqdn
		wantEndPoints  []endPoint
		wantPreference sbiaddr.Preference
	}{
		{
			name:          "IPv4",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4},
			wantIPv4:      testIPv4,
			wantURI:       testV4URI,
			wantEndPoints: []endPoint{{ipv4: testIPv4, port: testPort}},
		},
		{
			name:          "FQDN goes to fqdn, endpoint keeps only the port",
			sbi:           factory.Sbi{RegisterIPv4: testFQDN},
			wantIPv4:      testFQDN,
			wantURI:       testFQDNURI,
			wantFqdn:      testFQDN,
			wantEndPoints: []endPoint{{port: testPort}},
		},
		{
			name:          "IPv6 under registerIPv4 goes to the IPv6 slot",
			sbi:           factory.Sbi{RegisterIPv4: testIPv6},
			wantIPv6:      testIPv6,
			wantURI:       testV6URI,
			wantEndPoints: []endPoint{{ipv6: testIPv6, port: testPort}},
		},
		{
			name:          "registerIPv6 only",
			sbi:           factory.Sbi{RegisterIPv6: "[FD00:0::5]"},
			wantIPv6:      testIPv6,
			wantURI:       testV6URI,
			wantEndPoints: []endPoint{{ipv6: testIPv6, port: testPort}},
		},
		{
			name:          "dual stack: two endpoints, IPv4 first, IPv4 URI by default",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       testV4URI,
			wantEndPoints: []endPoint{{ipv4: testIPv4, port: testPort}, {ipv6: testIPv6, port: testPort}},
		},
		{
			name:           "dual stack with preferredFamily IPv6",
			sbi:            factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6, PreferredFamily: "IPv6"},
			wantIPv4:       testIPv4,
			wantIPv6:       testIPv6,
			wantURI:        testV6URI,
			wantEndPoints:  []endPoint{{ipv4: testIPv4, port: testPort}, {ipv6: testIPv6, port: testPort}},
			wantPreference: sbiaddr.PreferIPv6,
		},
		{
			name:          "unknown preferredFamily falls back to IPv4",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6, PreferredFamily: "IPv5"},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       testV4URI,
			wantEndPoints: []endPoint{{ipv4: testIPv4, port: testPort}, {ipv6: testIPv6, port: testPort}},
		},
		{
			name:          "FQDN plus IPv6",
			sbi:           factory.Sbi{RegisterIPv4: testFQDN, RegisterIPv6: testIPv6},
			wantIPv4:      testFQDN,
			wantIPv6:      testIPv6,
			wantURI:       testFQDNURI,
			wantFqdn:      testFQDN,
			wantEndPoints: []endPoint{{ipv6: testIPv6, port: testPort}},
		},
		{
			name:          "POD_IP holding IPv6 under registerIPv4",
			env:           map[string]string{sbiaddr.EnvPodIP: testPodIPv6},
			sbi:           factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP},
			wantIPv6:      testPodIPv6,
			wantURI:       "https://[" + testPodIPv6 + "]:29503",
			wantEndPoints: []endPoint{{ipv6: testPodIPv6, port: testPort}},
		},
		{
			name:          "POD_IP holding IPv4 under registerIPv4",
			env:           map[string]string{sbiaddr.EnvPodIP: testIPv4},
			sbi:           factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP},
			wantIPv4:      testIPv4,
			wantURI:       testV4URI,
			wantEndPoints: []endPoint{{ipv4: testIPv4, port: testPort}},
		},
		{
			name:          "POD_IPS dual stack",
			env:           map[string]string{sbiaddr.EnvPodIPs: "fd00::5,10.0.0.5"},
			sbi:           factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIPs},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       testV4URI,
			wantEndPoints: []endPoint{{ipv4: testIPv4, port: testPort}, {ipv6: testIPv6, port: testPort}},
		},
		{
			name:          "nothing configured: the UDM default",
			sbi:           factory.Sbi{},
			wantIPv4:      factory.UDM_DEFAULT_IPV4,
			wantURI:       testDefault,
			wantEndPoints: []endPoint{{ipv4: factory.UDM_DEFAULT_IPV4, port: testPort}},
		},
		{
			name:          "unset POD_IP: the UDM default",
			sbi:           factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP},
			wantIPv4:      factory.UDM_DEFAULT_IPV4,
			wantURI:       testDefault,
			wantEndPoints: []endPoint{{ipv4: factory.UDM_DEFAULT_IPV4, port: testPort}},
		},
		{
			name:          "invalid value: the UDM default",
			sbi:           factory.Sbi{RegisterIPv4: "10.0.0.256"},
			wantIPv4:      factory.UDM_DEFAULT_IPV4,
			wantURI:       testDefault,
			wantEndPoints: []endPoint{{ipv4: factory.UDM_DEFAULT_IPV4, port: testPort}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(sbiaddr.EnvPodIP, "")
			t.Setenv(sbiaddr.EnvPodIPs, "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			sbi := tt.sbi
			sbi.Scheme = "https"
			sbi.Port = testPort
			ctx := initWithSbi(t, &sbi)

			if ctx.RegisterIPv4 != tt.wantIPv4 || ctx.RegisterIPv6 != tt.wantIPv6 {
				t.Errorf("RegisterIPv4, RegisterIPv6 = %q, %q, want %q, %q",
					ctx.RegisterIPv4, ctx.RegisterIPv6, tt.wantIPv4, tt.wantIPv6)
			}
			if ctx.SbiPreference != tt.wantPreference {
				t.Errorf("SbiPreference = %v, want %v", ctx.SbiPreference, tt.wantPreference)
			}
			if got := ctx.GetSbiUri(); got != tt.wantURI {
				t.Errorf("GetSbiUri() = %q, want %q", got, tt.wantURI)
			}
			if got := ctx.GetIPv4Uri(); got != tt.wantURI {
				t.Errorf("GetIPv4Uri() = %q, want %q", got, tt.wantURI)
			}
			if u, err := url.Parse(ctx.GetSbiUri()); err != nil || u.Port() != "29503" {
				t.Errorf("url.Parse(%q) = %v, %v; want port 29503", ctx.GetSbiUri(), u, err)
			}

			if len(ctx.NfService) != 2 {
				t.Fatalf("expected 2 NF services, got %d", len(ctx.NfService))
			}
			for name, svc := range ctx.NfService {
				if got := svc.GetApiPrefix(); got != tt.wantURI {
					t.Errorf("%s apiPrefix = %q, want %q", name, got, tt.wantURI)
				}
				if got := svc.GetFqdn(); got != tt.wantFqdn {
					t.Errorf("%s fqdn = %q, want %q", name, got, tt.wantFqdn)
				}
				got := endPoints(svc)
				if len(got) != len(tt.wantEndPoints) {
					t.Fatalf("%s endpoints = %+v, want %+v", name, got, tt.wantEndPoints)
				}
				for i := range got {
					if got[i] != tt.wantEndPoints[i] {
						t.Errorf("%s endpoint %d = %+v, want %+v", name, i, got[i], tt.wantEndPoints[i])
					}
				}
			}
		})
	}
}

func TestInitUDMContextBindingAddr(t *testing.T) {
	const bindEnv = "UDM_TEST_BIND"
	tests := []struct {
		name string
		env  string
		sbi  factory.Sbi
		want string
	}{
		{name: "default", want: sbiaddr.DefaultBindHost},
		{name: "bindingIPv4 IPv4", sbi: factory.Sbi{BindingIPv4: testAnyV4}, want: testAnyV4},
		{name: "bindingIPv4 IPv6 wildcard", sbi: factory.Sbi{BindingIPv4: "::"}, want: "::"},
		{name: "bindingAddr", sbi: factory.Sbi{BindingAddr: "::"}, want: "::"},
		{name: "bindingAddr wins over bindingIPv4", sbi: factory.Sbi{BindingAddr: testIPv6, BindingIPv4: testAnyV4}, want: testIPv6},
		{name: "bracketed bindingAddr", sbi: factory.Sbi{BindingAddr: "[fd00::5]"}, want: testIPv6},
		{name: "bindingIPv4 from env", env: testIPv6, sbi: factory.Sbi{BindingIPv4: bindEnv}, want: testIPv6},
		{name: "bindingAddr from env", env: testIPv4, sbi: factory.Sbi{BindingAddr: bindEnv}, want: testIPv4},
		{name: "a list falls back to the wildcard", env: "10.0.0.5,fd00::5", sbi: factory.Sbi{BindingAddr: bindEnv}, want: sbiaddr.DefaultBindHost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(bindEnv, tt.env)
			sbi := tt.sbi
			ctx := initWithSbi(t, &sbi)
			if ctx.BindingIPv4 != tt.want {
				t.Errorf("BindingIPv4 = %q, want %q", ctx.BindingIPv4, tt.want)
			}
		})
	}
}

func TestSearchNFServiceUriAddressFamilies(t *testing.T) {
	https := models.URISCHEME_HTTPS
	newService := func(endPoints ...models.IpEndPoint) models.NFService {
		svc := models.NewNFService("0", models.SERVICENAME_NUDR_DR, nil, https, models.NFSERVICESTATUS_REGISTERED)
		svc.SetIpEndPoints(endPoints)
		return *svc
	}
	v4 := func(addr string) models.IpEndPoint {
		ep := models.NewIpEndPoint()
		ep.SetIpv4Address(addr)
		ep.SetPort(29504)
		return *ep
	}
	v6 := func(addr string) models.IpEndPoint {
		ep := models.NewIpEndPoint()
		ep.SetIpv6Address(addr)
		ep.SetPort(29504)
		return *ep
	}
	portOnly := models.NewIpEndPoint()
	portOnly.SetPort(29504)
	const udrV6URI = "https://[fd00::4]:29504"

	tests := []struct {
		name       string
		fqdn       string
		service    models.NFService
		preference sbiaddr.Preference
		want       string
	}{
		{name: "IPv4 endpoint", service: newService(v4("10.0.0.4")), want: "https://10.0.0.4:29504"},
		{name: "IPv6-only endpoint is bracketed", service: newService(v6("fd00::4")), want: udrV6URI},
		{name: "dual stack, IPv4 preferred", service: newService(v4("10.0.0.4"), v6("fd00::4")), want: "https://10.0.0.4:29504"},
		{
			name: "dual stack, IPv6 preferred", service: newService(v4("10.0.0.4"), v6("fd00::4")),
			preference: sbiaddr.PreferIPv6, want: udrV6URI,
		},
		{name: "profile FQDN with the endpoint port", fqdn: "udr", service: newService(*portOnly), want: "https://udr:29504"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := context.UDM_Self()
			orig := self.SbiPreference
			t.Cleanup(func() { self.SbiPreference = orig })
			self.SbiPreference = tt.preference

			profile := models.NewNFProfileDiscoveryWithDefaults()
			if tt.fqdn != "" {
				profile.SetFqdn(tt.fqdn)
			}
			profile.SetNfServices([]models.NFService{tt.service})
			got := SearchNFServiceUri(*profile, models.SERVICENAME_NUDR_DR, models.NFSERVICESTATUS_REGISTERED)
			if got != tt.want {
				t.Errorf("SearchNFServiceUri() = %q, want %q", got, tt.want)
			}
		})
	}
}
