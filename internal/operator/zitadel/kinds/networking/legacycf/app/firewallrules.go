package app

import (
	"reflect"

	"github.com/caos/orbos/internal/operator/zitadel/kinds/networking/legacycf/cloudflare"
)

func (a *App) EnsureFirewallRules(domain string, rules []*cloudflare.FirewallRule) ([]*cloudflare.FirewallRule, error) {
	result := make([]*cloudflare.FirewallRule, 0)
	currentRules, err := a.cloudflare.GetFirewallRules(domain)
	if err != nil {
		return nil, err
	}

	deleteRules := getFirewallRulesToDelete(currentRules, rules, a.TrimInternalPrefix)
	if len(deleteRules) > 0 {
		if err := a.cloudflare.DeleteFirewallRules(domain, deleteRules); err != nil {
			return nil, err
		}
	}

	createRules := getFirewallRulesToCreate(currentRules, rules, a.TrimInternalPrefix)
	if len(createRules) > 0 {
		created, err := a.cloudflare.CreateFirewallRules(domain, createRules)
		if err != nil {
			return nil, err
		}

		result = append(result, created...)
	}

	updateRules := getFirewallRulesToUpdate(currentRules, rules, a.TrimInternalPrefix)
	if len(updateRules) > 0 {
		updated, err := a.cloudflare.UpdateFirewallRules(domain, updateRules)
		if err != nil {
			return nil, err
		}

		result = append(result, updated...)
	}

	return result, nil
}

func getFirewallRulesToDelete(currentRules []*cloudflare.FirewallRule, rules []*cloudflare.FirewallRule, trimInternalPrefix func(string) string) []string {
	deleteRules := make([]string, 0)

	for _, currentRule := range currentRules {
		found := false
		for _, rule := range rules {
			if currentRule.Description == rule.Description {
				found = true
			}
		}

		if found == false {
			deleteRules = append(deleteRules, currentRule.ID)
		}
	}

	return deleteRules
}

func getFirewallRulesToCreate(currentRules []*cloudflare.FirewallRule, rules []*cloudflare.FirewallRule, trimInternalPrefix func(string) string) []*cloudflare.FirewallRule {
	createRules := make([]*cloudflare.FirewallRule, 0)

	for _, rule := range rules {
		found := false
		for _, currentRule := range currentRules {
			if currentRule.Description == rule.Description {
				found = true
				break
			}
		}
		if found == false {
			createRules = append(createRules, rule)
		}
	}

	return createRules
}

func getFirewallRulesToUpdate(currentRules []*cloudflare.FirewallRule, rules []*cloudflare.FirewallRule, trimInternalPrefix func(string) string) []*cloudflare.FirewallRule {
	updateRules := make([]*cloudflare.FirewallRule, 0)

	for _, rule := range rules {
		for _, currentRule := range currentRules {
			if currentRule.Description == rule.Description &&
				!reflect.DeepEqual(currentRule, rule) {
				rule.ID = currentRule.ID
				updateRules = append(updateRules, rule)
			}
		}
	}

	return updateRules
}
