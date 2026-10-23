// Author: Arthur
package handlers_test

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	profilev1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1/profilev1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestAPlayerCardShowsWhereTheyStand: the card another player opens says exactly what the
// player's own profile says (points, rank, stats and every badge, earned or locked) and adds their
// plates newest first. A card worked out from the wrong history would disagree with the profile.
func TestAPlayerCardShowsWhereTheyStand(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	ben := profileOf(t, srv, "test_ben")
	low := scannedAt(t, srv, "test_ben", 29, profileMonday.Add(9*time.Hour))
	high := scannedAt(t, srv, "test_ben", 91, profileMonday.AddDate(0, 0, 1).Add(9*time.Hour))
	platedFrom(t, srv, "test_ben", low, profileMonday.Add(12*time.Hour), "monday",
		award(repository.ReasonCook, 100), award(repository.ReasonAgainstAllOdds, 100), award(repository.ReasonStreak, 10))
	platedFrom(t, srv, "test_ben", high, profileMonday.AddDate(0, 0, 1).Add(12*time.Hour), "tuesday",
		award(repository.ReasonCook, 100), award(repository.ReasonStreak, 20))
	platedFrom(t, srv, "test_ben", high, profileMonday.AddDate(0, 0, 2).Add(12*time.Hour), "wednesday",
		award(repository.ReasonCook, 100), award(repository.ReasonStreak, 30))
	ranOnce(t, srv, "test_ben", profileMonday.AddDate(0, 0, -4).Add(12*time.Hour))

	card := getCard(t, maya, "test_ben")
	own := getProfile(t, ben)

	if card.GetHandle() != "test_ben" || card.GetAvatar() != "🍟" || card.GetPoints() != 460 {
		t.Errorf("card = %s %s with %d points, want test_ben 🍟 with 460", card.GetHandle(), card.GetAvatar(), card.GetPoints())
	}
	if !proto.Equal(card.GetRank(), own.GetRank()) {
		t.Errorf("card rank = %v, want the profile's %v", card.GetRank(), own.GetRank())
	}
	if !proto.Equal(card.GetStats(), own.GetStats()) {
		t.Errorf("card stats = %v, want the profile's %v", card.GetStats(), own.GetStats())
	}
	if len(card.GetBadges()) != 11 {
		t.Fatalf("card has %d badges, want all eleven", len(card.GetBadges()))
	}
	for i, b := range card.GetBadges() {
		if !proto.Equal(b, own.GetBadges()[i]) {
			t.Errorf("card badge %d = %v, want the profile's %v", i, b, own.GetBadges()[i])
		}
	}
	var plates []string
	for _, p := range card.GetRecentPlates() {
		plates = append(plates, fmt.Sprintf("%s %s %s",
			p.GetPlatedAt().AsTime().In(points.Campus()).Format("Mon"), p.GetRecipeTitle(), p.GetPhoto()[4:]))
	}
	want := []string{"Wed Egg fried rice wednesday", "Tue Egg fried rice tuesday", "Mon Egg fried rice monday"}
	if !slices.Equal(plates, want) {
		t.Errorf("plates = %v, want %v", plates, want)
	}
}

// TestACardCarriesTheNewestSixPlates: a card is one sheet, so it carries six plates and no cursor.
// Of seven, the oldest is the one left off, so the card never shows a stale corner of a history.
func TestACardCarriesTheNewestSixPlates(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	profileOf(t, srv, "test_ben")
	s := scannedAt(t, srv, "test_ben", 70, profileMonday)
	for i := range 7 {
		// Each plate in a meal window of its own, six hours apart.
		platedFrom(t, srv, "test_ben", s, profileMonday.Add(time.Duration(6*i+6)*time.Hour), fmt.Sprint("plate ", i),
			award(repository.ReasonCook, 100))
	}

	card := getCard(t, maya, "test_ben")

	var got []string
	for _, p := range card.GetRecentPlates() {
		got = append(got, string(p.GetPhoto()[4:]))
	}
	want := []string{"plate 6", "plate 5", "plate 4", "plate 3", "plate 2", "plate 1"}
	if !slices.Equal(got, want) {
		t.Errorf("plates = %v, want the newest six, newest first: %v", got, want)
	}
}

// TestThePlayerCardCarriesNothingPrivate: a card is shown to anyone signed in, so it must never
// carry a player's kitchen settings. The message has no field that could hold them, at any depth,
// and a card of a player whose kitchen is set spells none of its words on the wire.
func TestThePlayerCardCarriesNothingPrivate(t *testing.T) {
	t.Parallel()
	private := []string{"kitchen", "diet", "allerg", "equipment", "staple", "roast"}
	for _, field := range fieldNames((&profilev1.PlayerCard{}).ProtoReflect().Descriptor()) {
		for _, word := range private {
			if strings.Contains(strings.ToLower(field), word) {
				t.Errorf("PlayerCard has the field %s, which could carry a kitchen setting (%s)", field, word)
			}
		}
	}

	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	profileOf(t, srv, "test_ben")
	_, err := repository.New(srv.pool).UpdateKitchen(t.Context(), accountIDOf(t, srv, "test_ben"), repository.KitchenEdit{
		RoastLevel: repository.RoastExtraCrispy,
		Equipment:  []string{repository.EquipmentAirFryer},
		Diet:       repository.DietPescatarian,
		Allergies:  []string{repository.AllergySesame, repository.AllergyShellfish},
		Staples:    []string{repository.StapleButter},
	})
	if err != nil {
		t.Fatalf("set ben's kitchen: %v", err)
	}

	wire, err := protojson.Marshal(getCard(t, maya, "test_ben"))
	if err != nil {
		t.Fatalf("encode the card: %v", err)
	}
	for _, word := range []string{"extra_crispy", "air_fryer", "pescatarian", "sesame", "shellfish", "butter"} {
		if strings.Contains(string(wire), word) {
			t.Errorf("ben's card carries %q from his kitchen: %s", word, wire)
		}
	}
}

// TestACardIsFoundByTheHandleAsTyped: a handle reads the same however it is cased or spaced, as it
// did when it was made, and a handle nobody holds is not_found rather than someone else's card.
func TestACardIsFoundByTheHandleAsTyped(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	profileOf(t, srv, "test_ben")

	card := getCard(t, maya, " Test_Ben ")
	_, err := maya.GetPlayerCard(t.Context(), connect.NewRequest(&profilev1.GetPlayerCardRequest{Handle: "test_nobody"}))

	if card.GetHandle() != "test_ben" {
		t.Errorf("card = %s, want test_ben", card.GetHandle())
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("got code %v (%v), want not_found", connect.CodeOf(err), err)
	}
}

// TestABadCardHandleIsRefused: a handle that could never have been made is refused on its field,
// with the words the sign-up form uses, and never reaches the database.
func TestABadCardHandleIsRefused(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	cases := []struct {
		name, handle, want string
	}{
		{name: "empty", handle: "  ", want: "handle is required"},
		{name: "too short", handle: "ab", want: "handle must be at least 3 characters"},
		{name: "a space inside", handle: "chef maya", want: "handle can only use lowercase letters, numbers and underscores"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := maya.GetPlayerCard(t.Context(), connect.NewRequest(&profilev1.GetPlayerCardRequest{Handle: c.handle}))

			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
			}
			got := fieldViolations(t, err)
			if !maps.Equal(got, map[string]string{"handle": c.want}) {
				t.Errorf("violations = %v, want handle: %q", got, c.want)
			}
		})
	}
}

// TestAPlayerCardNeedsAnAccount: cards are for players, so a fresh install gets unauthenticated,
// which the app reads as "back to onboarding".
func TestAPlayerCardNeedsAnAccount(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	profileOf(t, srv, "test_ben")
	stranger := profilev1connect.NewProfileServiceClient(&http.Client{Jar: newcomer(t, srv.url).jar}, srv.url)

	_, err := stranger.GetPlayerCard(t.Context(), connect.NewRequest(&profilev1.GetPlayerCardRequest{Handle: "test_ben"}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("got code %v (%v), want unauthenticated", connect.CodeOf(err), err)
	}
}

func getCard(t *testing.T, c profilev1connect.ProfileServiceClient, handle string) *profilev1.PlayerCard {
	t.Helper()
	resp, err := c.GetPlayerCard(t.Context(), connect.NewRequest(&profilev1.GetPlayerCardRequest{Handle: handle}))
	if err != nil {
		t.Fatalf("get %q's card: %v", handle, err)
	}
	return resp.Msg.GetCard()
}

// fieldNames is every field of a message and of each message it holds, at any depth, as
// "Message.field".
func fieldNames(md protoreflect.MessageDescriptor) []string {
	var out []string
	seen := map[protoreflect.FullName]bool{}
	var walk func(protoreflect.MessageDescriptor)
	walk = func(md protoreflect.MessageDescriptor) {
		if seen[md.FullName()] {
			return
		}
		seen[md.FullName()] = true
		fields := md.Fields()
		for i := range fields.Len() {
			f := fields.Get(i)
			out = append(out, string(md.Name())+"."+string(f.Name()))
			if f.Message() != nil {
				walk(f.Message())
			}
		}
	}
	walk(md)
	return out
}
