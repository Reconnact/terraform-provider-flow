package flow

import (
	"context"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// goclient tags these fields `omitempty`, which drops false and 0 from the request:
// sticky sessions could never be switched off, a public router never made private,
// and an icmp rule with type or code 0 was rejected. The bodies below
// carry pointers and go through the raw client instead: nil is left out, a zero value is sent.
//
// TODO: remove this file once goclient has pointer fields on these structs, and go back to its
// typed services in the pool, router and both security group rule resources.

const (
	computeLoadBalancersPath       = "/v4/compute/load-balancers"
	computeRoutersPath             = "/v4/compute/routers"
	computeSecurityGroupsPath      = "/v4/compute/security-groups"
	macBareMetalSecurityGroupsPath = "/v4/macbaremetal/security-groups"
)

type loadBalancerPoolUpdateBody struct {
	CertificateID        *int                                    `json:"certificate_id,omitempty"`
	BalancingAlgorithmID *int                                    `json:"balancing_algorithm_id,omitempty"`
	StickySession        *bool                                   `json:"sticky_session,omitempty"`
	HealthCheck          *compute.LoadBalancerHealthCheckOptions `json:"health_check,omitempty"`
}

type routerUpdateBody struct {
	Name   string `json:"name,omitempty"`
	Public *bool  `json:"public,omitempty"`
}

type securityGroupRuleBody struct {
	Direction             string `json:"direction"`
	Protocol              int    `json:"protocol"`
	FromPort              int    `json:"from_port,omitempty"`
	ToPort                int    `json:"to_port,omitempty"`
	ICMPType              *int   `json:"icmp_type,omitempty"`
	ICMPCode              *int   `json:"icmp_code,omitempty"`
	IPRange               string `json:"ip_range,omitempty"`
	RemoteSecurityGroupID int    `json:"remote_security_group_id,omitempty"`
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueBool()
	return &v
}

func intPointer(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := int(value.ValueInt64())
	return &v
}

func updateLoadBalancerPool(ctx context.Context, client goclient.Client, loadBalancerID, poolID int, body loadBalancerPoolUpdateBody) (pool compute.LoadBalancerPool, err error) {
	err = client.Update(ctx, goclient.Join(computeLoadBalancersPath, loadBalancerID, "balancing-pools", poolID), body, &pool)
	return
}

func updateRouter(ctx context.Context, client goclient.Client, routerID int, body routerUpdateBody) (router compute.Router, err error) {
	err = client.Update(ctx, goclient.Join(computeRoutersPath, routerID), body, &router)
	return
}

func createComputeSecurityGroupRule(ctx context.Context, client goclient.Client, securityGroupID int, body securityGroupRuleBody) (rule compute.SecurityGroupRule, err error) {
	err = client.Create(ctx, goclient.Join(computeSecurityGroupsPath, securityGroupID, "rules"), body, &rule)
	return
}

func updateComputeSecurityGroupRule(ctx context.Context, client goclient.Client, securityGroupID, ruleID int, body securityGroupRuleBody) (rule compute.SecurityGroupRule, err error) {
	err = client.Update(ctx, goclient.Join(computeSecurityGroupsPath, securityGroupID, "rules", ruleID), body, &rule)
	return
}

func createMacBareMetalSecurityGroupRule(ctx context.Context, client goclient.Client, securityGroupID int, body securityGroupRuleBody) (rule macbaremetal.SecurityGroupRule, err error) {
	err = client.Create(ctx, goclient.Join(macBareMetalSecurityGroupsPath, securityGroupID, "rules"), body, &rule)
	return
}

func updateMacBareMetalSecurityGroupRule(ctx context.Context, client goclient.Client, securityGroupID, ruleID int, body securityGroupRuleBody) (rule macbaremetal.SecurityGroupRule, err error) {
	err = client.Update(ctx, goclient.Join(macBareMetalSecurityGroupsPath, securityGroupID, "rules", ruleID), body, &rule)
	return
}
