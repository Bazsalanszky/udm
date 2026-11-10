// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"testing"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/util/sbiaddr"
)

const (
	uriTestIPv4 = "10.0.0.5"
	uriTestIPv6 = "fd00::5"
	uriTestSupi = "imsi-001010000000001"
)

func TestLocationURIsBracketIPv6(t *testing.T) {
	tests := []struct {
		name         string
		registerIPv4 string
		registerIPv6 string
		preference   sbiaddr.Preference
		wantBase     string
	}{
		{name: "IPv4", registerIPv4: uriTestIPv4, wantBase: "https://10.0.0.5:29503"},
		{name: "FQDN", registerIPv4: "udm", registerIPv6: uriTestIPv6, wantBase: "https://udm:29503"},
		{name: "IPv6", registerIPv6: uriTestIPv6, wantBase: "https://[fd00::5]:29503"},
		{name: "dual stack, IPv4 preferred", registerIPv4: uriTestIPv4, registerIPv6: uriTestIPv6, wantBase: "https://10.0.0.5:29503"},
		{
			name: "dual stack, IPv6 preferred", registerIPv4: uriTestIPv4, registerIPv6: uriTestIPv6,
			preference: sbiaddr.PreferIPv6, wantBase: "https://[fd00::5]:29503",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := UDM_Self()
			origV4, origV6, origPref := self.RegisterIPv4, self.RegisterIPv6, self.SbiPreference
			origScheme, origPort := self.UriScheme, self.SBIPort
			t.Cleanup(func() {
				self.RegisterIPv4, self.RegisterIPv6, self.SbiPreference = origV4, origV6, origPref
				self.UriScheme, self.SBIPort = origScheme, origPort
			})
			self.RegisterIPv4, self.RegisterIPv6, self.SbiPreference = tt.registerIPv4, tt.registerIPv6, tt.preference
			self.UriScheme, self.SBIPort = models.URISCHEME_HTTPS, 29503

			ue := &UdmUeContext{Supi: uriTestSupi, PduSessionID: "5"}
			checks := map[string]string{
				ue.GetLocationURI(LocationUriAmf3GppAccessRegistration): tt.wantBase +
					"/nudm-uecm/v1/" + uriTestSupi + "/registrations/amf-3gpp-access",
				ue.GetLocationURI(LocationUriSmfRegistration): tt.wantBase +
					"/nudm-uecm/v1/" + uriTestSupi + "/registrations/smf-registrations/5",
				ue.GetLocationURI2(LocationUriSdmSubscription, uriTestSupi): tt.wantBase +
					"/nudm-sdm/v2/" + uriTestSupi + "/sdm-subscriptions/",
				self.GetSDMUri(): tt.wantBase + "/nudm-sdm/v2",
			}
			for got, want := range checks {
				if got != want {
					t.Errorf("got %q, want %q", got, want)
				}
			}
		})
	}
}

// TestGetSbiUriKeepsLegacyFormat checks that, for the RegisterIPv4 values that
// worked before (IPv4 addresses and host names), the own URI is the same as
// the one built with fmt.Sprintf("%s://%s:%d").
func TestGetSbiUriKeepsLegacyFormat(t *testing.T) {
	for _, host := range []string{"127.0.0.3", "1.1.1.1", "udm", "udm.sdcore.svc.cluster.local"} {
		for _, scheme := range []models.UriScheme{"", models.URISCHEME_HTTP, models.URISCHEME_HTTPS} {
			for _, port := range []int{80, 8000, 29503} {
				ctx := &UDMContext{UriScheme: scheme, RegisterIPv4: host, SBIPort: port}
				legacy := fmt.Sprintf("%s://%s:%d", scheme, host, port)
				if got := ctx.GetSbiUri(); got != legacy {
					t.Errorf("GetSbiUri() = %q, want %q", got, legacy)
				}
			}
		}
	}
}
