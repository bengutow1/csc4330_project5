// Author: Caden Lund
package fries

import (
	friesv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1"
)

// toProtoListSpots maps a page onto the wire.
func toProtoListSpots(p page) *friesv1.ListSpotsResponse {
	out := &friesv1.ListSpotsResponse{Spots: make([]*friesv1.Spot, 0, len(p.spots))}
	for _, s := range p.spots {
		out.Spots = append(out.Spots, toProtoSpot(s))
	}
	if p.next != nil {
		out.NextCursor = encodeCursor(*p.next)
	}
	if p.nearest != nil {
		out.NearestFries = toProtoSpot(*p.nearest)
	}
	return out
}

// toProtoSpot maps one spot onto the wire.
func toProtoSpot(s spot) *friesv1.Spot {
	return &friesv1.Spot{
		SpotId: s.ID,
		Name:   s.Name,
		// No two points on Earth are more than 20,040 km apart, well inside int32.
		DistanceMeters: int32(s.DistanceMeters),
		OpenNow:        s.OpenNow,
		ClosesAt:       s.ClosesAt,
		LateNight:      s.lateNight,
		PriceLevel:     int32(s.PriceLevel),
		FastFood:       s.FastFood,
		Walkable:       s.walkable,
		Lat:            s.Lat,
		Lng:            s.Lng,
		Address:        s.Address,
		Category:       s.Category,
	}
}
