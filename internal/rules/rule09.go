package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule09: duplicate event_id across >=2 entries is an error when the group
// has no mutual related pair (a bidirectional cross-link between two members).
func rule09(c *Context) {
	for _, eid := range sortedKeys(c.EventGroups) {
		members := c.EventGroups[eid]
		if len(members) < 2 {
			continue
		}
		if hasMutualLink(members, c.RelatedLinks) {
			continue
		}
		c.Add(9, vault.SeverityError, firstMemberPath(members), fmt.Sprintf("duplicate event_id %s across %d entries with no mutual related links between them", eid, len(members)))
	}
}

func hasMutualLink(members []*EntryInfo, links map[string]map[string]bool) bool {
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			a, b := members[i].File.Path, members[j].File.Path
			if links[a][b] && links[b][a] {
				return true
			}
		}
	}
	return false
}

func firstMemberPath(members []*EntryInfo) string {
	p := members[0].File.Path
	for _, m := range members[1:] {
		if m.File.Path < p {
			p = m.File.Path
		}
	}
	return p
}
