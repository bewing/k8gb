package utils

/*
Copyright 2022 The k8gb Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Generated by GoLic, for more details see: https://github.com/AbsaOSS/golic
*/

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

type DNSServer struct {
	Host string
	Port int
}

type DNSList []DNSServer

func (s DNSServer) String() string {
	return fmt.Sprintf("%s:%v", s.Host, s.Port)
}

func (l DNSList) String() string {
	var aux []string
	for _, el := range l {
		aux = append(aux, el.String())
	}
	return strings.Join(aux, ",")
}

// Dig returns a list of IP addresses for a given FQDN by using the dns servers from edgeDNSServers
// dns servers are tried one by one from the edgeDNSServers and if there is a non-error response it is returned and the rest is not tried
func Dig(fqdn string, edgeDNSServers ...DNSServer) (ips []string, err error) {
	if len(edgeDNSServers) == 0 {
		return nil, fmt.Errorf("empty edgeDNSServers, provide at least one")
	}
	if len(fqdn) == 0 {
		return
	}

	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}
	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, dns.TypeA)
	ack, err := Exchange(msg, edgeDNSServers)
	if err != nil {
		return nil, fmt.Errorf("dig error: %s", err)
	}
	aRecords := make([]*dns.A, 0)
	cnameRecords := make([]*dns.CNAME, 0)
	for _, a := range ack.Answer {
		switch v := a.(type) {
		case *dns.A:
			ips = append(ips, v.A.String())
			aRecords = append(aRecords, v)
		case *dns.CNAME:
			cnameRecords = append(cnameRecords, v)
		}
	}
	resolved := func(c *dns.CNAME) bool {
		for _, a := range aRecords {
			if c.Target == a.A.String() {
				return true
			}
		}
		return false
	}
	// Check for non-resolved CNAMEs
	for _, cname := range cnameRecords {
		if !resolved(cname) {
			cnameIPs, err := Dig(cname.Target, edgeDNSServers...)
			if err != nil {
				return nil, err
			}
			ips = append(ips, cnameIPs...)
		}
	}
	sort.Strings(ips)
	return
}

func Exchange(m *dns.Msg, edgeDNSServers []DNSServer) (msg *dns.Msg, err error) {
	if len(edgeDNSServers) == 0 {
		return nil, fmt.Errorf("empty edgeDNSServers, provide at least one")
	}
	for _, ns := range edgeDNSServers {
		if ns.Host == "" {
			return nil, fmt.Errorf("empty edgeDNSServer.Host in the list")
		}
		msg, err = dns.Exchange(m, ns.String())
		if err != nil {
			continue
		}
		return
	}
	return nil, fmt.Errorf("exchange error: all dns servers were tried and none of them were able to resolve, err: %s", err)
}
