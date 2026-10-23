// Author: LAPTOP-BPMQJP10\andyk
package combo

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	combov1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
)

func toProtoCombo(c repository.Combo) (*combov1.Combo, error) {
	out := &combov1.Combo{ComboId: c.ComboID, Code: c.Code, HostId: c.HostID, ExpiresAt: timestamppb.New(c.ExpiresAt)}
	switch c.State {
	case repository.ComboOpen:
		out.State = combov1.ComboState_COMBO_STATE_OPEN
	case repository.ComboEnded:
		out.State = combov1.ComboState_COMBO_STATE_ENDED
	default:
		return nil, fmt.Errorf("combo: unknown state %q", c.State)
	}
	ready := len(c.Members) == 2
	var merged []repository.ScanItem
	var owned []*combov1.OwnedIngredient
	for _, m := range c.Members {
		p := &combov1.Player{AccountId: m.AccountID, Handle: m.Handle, Avatar: m.Avatar}
		if m.Scan == nil {
			ready = false
		} else {
			var err error
			p.Scan, err = handlers.ToProtoVerdict(*m.Scan)
			if err != nil {
				return nil, err
			}
			merged = append(merged, m.Scan.Items...)
			for _, item := range p.Scan.Ingredients {
				owned = append(owned, &combov1.OwnedIngredient{AccountId: m.AccountID, Ingredient: item})
			}
		}
		out.Players = append(out.Players, p)
	}
	if ready {
		items, err := handlers.ScoringItems(merged)
		if err != nil {
			return nil, err
		}
		score := scoring.Fridge(items)
		tier, err := handlers.WireTier(scoring.TierOf(score))
		if err != nil {
			return nil, err
		}
		out.Verdict = &combov1.CombinedVerdict{Score: int32(score), Tier: tier, Ingredients: owned}
	}
	return out, nil
}
