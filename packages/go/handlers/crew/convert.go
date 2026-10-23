// Author: Ben Gutowski
package crew

import (
	crewv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/crew/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// toProtoCrew maps a crew as a member sees it to the wire.
func toProtoCrew(c repository.Crew) *crewv1.Crew {
	return &crewv1.Crew{
		CrewId:      c.CrewID,
		Name:        c.Name,
		Code:        c.Code,
		MemberCount: int32(c.Members), //nolint:gosec // a crew's members fit an int32 many times over
	}
}

// toProtoCrews maps a player's crews, in the order the repository gave.
func toProtoCrews(cs []repository.Crew) []*crewv1.Crew {
	out := make([]*crewv1.Crew, 0, len(cs))
	for _, c := range cs {
		out = append(out, toProtoCrew(c))
	}
	return out
}
