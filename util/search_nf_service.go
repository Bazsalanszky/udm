// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package util

import (
	"github.com/omec-project/openapi/v2/models"
	openapiutils "github.com/omec-project/openapi/v2/utils"
	"github.com/omec-project/udm/context"
	"github.com/omec-project/util/sbiaddr"
)

// SearchNFServiceUri returns the base URI (scheme://host:port, IPv6 in
// brackets, or the service apiPrefix) of the first service of nfProfile with
// the given name and status, or "" when none yields one. Of a dual-stack
// peer, the address family set by sbi.preferredFamily is used.
func SearchNFServiceUri(nfProfile models.NFProfileDiscovery, serviceName models.ServiceName,
	nfServiceStatus models.NFServiceStatus,
) string {
	return openapiutils.SearchNFServiceURI(&nfProfile, serviceName, nfServiceStatus,
		context.UDM_Self().SbiPreference == sbiaddr.PreferIPv6)
}
