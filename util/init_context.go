// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-FileCopyrightText: 2024 Canonical Ltd.
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0
//

package util

import (
	"cmp"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/udm/context"
	"github.com/omec-project/udm/factory"
	"github.com/omec-project/udm/logger"
	"github.com/omec-project/udm/suci"
	"github.com/omec-project/util/sbiaddr"
)

// InitUDMContext initializes the UDM context with configuration data.
func InitUDMContext(udmContext *context.UDMContext) {
	config := factory.UdmConfig
	logger.UtilLog.Info("udmconfig Info: Version[", config.Info.Version, "] Description[", config.Info.Description, "]")
	configuration := config.Configuration
	udmContext.NfId = uuid.New().String()
	if configuration.UdmName != "" {
		udmContext.Name = configuration.UdmName
	}

	// Initialize default SBI values
	udmContext.SBIPort = factory.UDM_DEFAULT_PORT_INT
	udmContext.RegisterIPv4 = factory.UDM_DEFAULT_IPV4
	udmContext.RegisterIPv6 = ""
	udmContext.SbiPreference = sbiaddr.PreferIPv4
	udmContext.UriScheme = ""

	// Refactored SBI and Binding logic
	initSbiSettings(udmContext, configuration.Sbi)

	// Refactored NRF caching logic
	initNrfCaching(udmContext, configuration)

	udmContext.NrfUri = configuration.NrfUri
	initSuciProfiles(udmContext, configuration)

	udmContext.InitNFService(configuration.ServiceList, config.Info.Version)
}

// initSbiSettings handles the SBI configuration and IP binding logic.
func initSbiSettings(udmContext *context.UDMContext, sbi *factory.Sbi) {
	if sbi == nil {
		return
	}

	if sbi.Scheme != "" {
		udmContext.UriScheme = models.UriScheme(sbi.Scheme)
	}
	initRegisterAddrs(udmContext, sbi)
	if sbi.Port != 0 {
		udmContext.SBIPort = sbi.Port
	}

	// Binding address: bindingAddr, else its legacy alias bindingIPv4; either
	// may name an environment variable. The default is 0.0.0.0.
	configured := cmp.Or(strings.TrimSpace(sbi.BindingAddr), strings.TrimSpace(sbi.BindingIPv4))
	switch {
	case configured == "":
		logger.UtilLog.Warn("Error parsing ServerIPv4 address as string. Using the 0.0.0.0 address as default.")
	case os.Getenv(configured) != "":
		logger.UtilLog.Info("Parsing ServerIPv4 address from ENV Variable.")
	}
	var err error
	if udmContext.BindingIPv4, err = sbiaddr.BindHost(sbi.BindingAddr, sbi.BindingIPv4); err != nil {
		logger.UtilLog.Warnf("SBI binding address: %v", err)
	}
}

// initRegisterAddrs sets RegisterIPv4 and RegisterIPv6 from the registerIPv4
// and registerIPv6 settings. Each may be an address of either family, a host
// name, or the name of an environment variable such as POD_IP or POD_IPS;
// every value goes to the field of its own kind. When nothing usable is
// configured, the UDM keeps its IPv4 default.
func initRegisterAddrs(udmContext *context.UDMContext, sbi *factory.Sbi) {
	addrs, err := sbiaddr.ResolveRegister(sbi.RegisterIPv4, sbi.RegisterIPv6)
	if err != nil {
		logger.UtilLog.Warnf("SBI register address: %v", err)
	}
	if addrs.Empty() {
		addrs.IPv4 = factory.UDM_DEFAULT_IPV4
	}
	if addrs.FQDN != "" && addrs.IPv4 != "" {
		logger.UtilLog.Warnf("SBI register address: registering FQDN %s, not IPv4 %s", addrs.FQDN, addrs.IPv4)
	}
	udmContext.RegisterIPv4 = cmp.Or(addrs.FQDN, addrs.IPv4)
	udmContext.RegisterIPv6 = addrs.IPv6

	if udmContext.SbiPreference, err = sbiaddr.ParsePreference(sbi.PreferredFamily); err != nil {
		logger.UtilLog.Warnf("SBI preferredFamily: %v; using IPv4", err)
	}
}

// initNrfCaching configures NRF caching and eviction intervals.
func initNrfCaching(udmContext *context.UDMContext, configuration *factory.Configuration) {
	if configuration == nil {
		return
	}
	udmContext.EnableNrfCaching = configuration.EnableNrfCaching
	if !udmContext.EnableNrfCaching {
		return
	}

	if configuration.NrfCacheEvictionInterval == 0 {
		udmContext.NrfCacheEvictionInterval = time.Duration(900) // Default: 15 mins
	} else {
		udmContext.NrfCacheEvictionInterval = time.Duration(configuration.NrfCacheEvictionInterval)
	}
}

// initSuciProfiles sets up the SUCI protection schemes.
func initSuciProfiles(udmContext *context.UDMContext, configuration *factory.Configuration) {
	if configuration == nil {
		return
	}
	udmContext.SuciProfiles = []suci.SuciProfile{
		{
			ProtectionScheme: "1", // Standard defined value for Protection Scheme A (TS 33.501 Annex C)
			PrivateKey:       configuration.Keys.UdmProfileAHNPrivateKey,
			PublicKey:        configuration.Keys.UdmProfileAHNPublicKey,
		},
		{
			ProtectionScheme: "2", // Standard defined value for Protection Scheme B (TS 33.501 Annex C)
			PrivateKey:       configuration.Keys.UdmProfileBHNPrivateKey,
			PublicKey:        configuration.Keys.UdmProfileBHNPublicKey,
		},
	}
}
