package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule13: perspective groups (same event_id, >=2 entries) must be a full
// directed mesh: every member's related points at every other member. Groups
// with no mutual link at all are reported by rule 9 only.
func rule13(c *Context) {
	for _, eid := range sortedKeys(c.EventGroups) {
		members := c.EventGroups[eid]
		if len(members) < 2 {
			continue
		}
		if !hasMutualLink(members, c.RelatedLinks) {
			continue
		}
		var edges []string
		for _, m := range members {
			for _, o := range members {
				if m == o {
					continue
				}
				if !c.RelatedLinks[m.File.Path][o.File.Path] {
					edges = append(edges, m.File.Path+" -> "+o.File.Path)
				}
			}
		}
		if len(edges) > 0 {
			sort.Strings(edges)
			c.Add(13, vault.SeverityError, firstMemberPath(members), fmt.Sprintf("perspective group %s must cross-reference every other member (full mesh); missing related: %s", eid, strings.Join(edges, "; ")))
		}
	}
}
