// Copyright (c) 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"strings"

	"github.com/vmware-tanzu/vm-operator/pkg/providers/vsphere/constants"
	"github.com/vmware-tanzu/vm-operator/pkg/util/netplan"
	"github.com/vmware-tanzu/vm-operator/pkg/util/ptr"
)

func NetPlanCustomization(result NetworkInterfaceResults) (*netplan.Network, error) {
	netPlan := &netplan.Network{
		Version:   constants.NetPlanVersion,
		Ethernets: make(map[string]netplan.Ethernet),
	}

	for _, r := range result.Results {
		npEth := netplan.Ethernet{
			Match: &netplan.Match{
				Macaddress: ptr.To(NormalizeNetplanMac(r.MacAddress)),
			},
			SetName: &r.GuestDeviceName,
			MTU:     &r.MTU,
			Nameservers: &netplan.Nameserver{
				Addresses: r.Nameservers,
				Search:    r.SearchDomains,
			},
		}

		npEth.Dhcp4 = &r.DHCP4
		npEth.Dhcp6 = &r.DHCP6

		if !*npEth.Dhcp4 {
			for i := range r.IPConfigs {
				ipConfig := r.IPConfigs[i]
				if ipConfig.IsIPv4 {
					if npEth.Gateway4 == nil || *npEth.Gateway4 == "" {
						npEth.Gateway4 = &ipConfig.Gateway
					}
					npEth.Addresses = append(
						npEth.Addresses,
						netplan.Address{
							String: &ipConfig.IPCIDR,
						},
					)
				}
			}
		}
		if !*npEth.Dhcp6 {
			for i := range r.IPConfigs {
				ipConfig := r.IPConfigs[i]
				if !ipConfig.IsIPv4 {
					if npEth.Gateway6 == nil || *npEth.Gateway6 == "" {
						npEth.Gateway6 = &ipConfig.Gateway
					}
					npEth.Addresses = append(
						npEth.Addresses,
						netplan.Address{
							String: &ipConfig.IPCIDR,
						},
					)
				}
			}
		}

		for i := range r.Routes {
			route := r.Routes[i]
			npEth.Routes = append(
				npEth.Routes,
				netplan.Route{
					To:     &route.To,
					Metric: ptr.To(int64(route.Metric)),
					Via:    &route.Via,
				},
			)
		}

		netPlan.Ethernets[r.Name] = npEth
	}

	return netPlan, nil
}

// NormalizeNetplanMac normalizes the mac address format to one compatible with netplan.
func NormalizeNetplanMac(mac string) string {
	mac = strings.ReplaceAll(mac, "-", ":")
	return strings.ToLower(mac)
}
