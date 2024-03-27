package scaleway

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"
)

// instanceAPIWithZone returns a new iam API for a Create request
func IamAPI(m interface{}) *iam.API {
	return iam.NewAPI(meta.ExtractScwClient(m))
}

func getOrganizationID(m interface{}, d *schema.ResourceData) *string {
	orgID, orgIDExist := d.GetOk("organization_id")

	if orgIDExist {
		return types.ExpandStringPtr(orgID)
	}

	defaultOrgID, defaultOrgIDExists := meta.ExtractScwClient(m).GetDefaultOrganizationID()
	if defaultOrgIDExists {
		return types.ExpandStringPtr(defaultOrgID)
	}

	return nil
}

func expandPermissionSetNames(rawPermissions interface{}) *[]string {
	permissions := []string{}
	permissionSet := rawPermissions.(*schema.Set)
	for _, rawPermission := range permissionSet.List() {
		permissions = append(permissions, rawPermission.(string))
	}
	return &permissions
}

func flattenPermissionSetNames(permissions []string) *schema.Set {
	rawPermissions := []interface{}(nil)
	for _, perm := range permissions {
		rawPermissions = append(rawPermissions, perm)
	}
	return schema.NewSet(func(i interface{}) int {
		return StringHashcode(i.(string))
	}, rawPermissions)
}

func expandPolicyRuleSpecs(d interface{}) []*iam.RuleSpecs {
	rules := []*iam.RuleSpecs(nil)
	rawRules := d.([]interface{})
	for _, rawRule := range rawRules {
		mapRule := rawRule.(map[string]interface{})
		rule := &iam.RuleSpecs{
			PermissionSetNames: expandPermissionSetNames(mapRule["permission_set_names"]),
		}
		if orgID, orgIDExists := mapRule["organization_id"]; orgIDExists && orgID.(string) != "" {
			rule.OrganizationID = scw.StringPtr(orgID.(string))
		}
		if projIDs, projIDsExists := mapRule["project_ids"]; projIDsExists {
			rule.ProjectIDs = types.ExpandStringsPtr(projIDs)
		}
		rules = append(rules, rule)
	}
	return rules
}

func flattenPolicyRules(rules []*iam.Rule) interface{} {
	rawRules := []interface{}(nil)
	for _, rule := range rules {
		rawRule := map[string]interface{}{}
		if rule.OrganizationID != nil {
			rawRule["organization_id"] = types.FlattenStringPtr(rule.OrganizationID)
		} else {
			rawRule["organization_id"] = nil
		}
		if rule.ProjectIDs != nil {
			rawRule["project_ids"] = types.FlattenSliceString(*rule.ProjectIDs)
		}
		if rule.PermissionSetNames != nil {
			rawRule["permission_set_names"] = flattenPermissionSetNames(*rule.PermissionSetNames)
		}
		rawRules = append(rawRules, rawRule)
	}
	return rawRules
}
