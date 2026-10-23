// Author: Ben Gutowski
package repository_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// crewCap is the cap these tests make crews under; the handler passes its own.
const crewCap = 3

// aCrewMaker makes an account named handle, for a test about crews.
func aCrewMaker(t *testing.T, repo *repository.Repository, handle string) int64 {
	t.Helper()
	a, err := repo.CreateAccount(t.Context(), handle, "🍟")
	if err != nil {
		t.Fatalf("create account %s: %v", handle, err)
	}
	return a.AccountID
}

// aCrew makes a crew for owner with code.
func aCrew(t *testing.T, repo *repository.Repository, owner int64, name, code string) repository.Crew {
	t.Helper()
	c, err := repo.CreateCrew(t.Context(), owner, name, code, crewCap)
	if err != nil {
		t.Fatalf("create crew %s: %v", name, err)
	}
	return c
}

// TestMakingACrewMakesItsOwnerItsFirstMember: a new crew must hold its maker, or their own crew
// would not be on their list.
func TestMakingACrewMakesItsOwnerItsFirstMember(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")

	made := aCrew(t, repo, maya, "Dorm 4B", "123456")

	want := repository.Crew{CrewID: made.CrewID, Name: "Dorm 4B", Code: "123456", Members: 1}
	if made != want {
		t.Errorf("made %+v, want %+v", made, want)
	}
	mine, err := repo.ListMyCrews(t.Context(), maya)
	if err != nil {
		t.Fatalf("list my crews: %v", err)
	}
	if len(mine) != 1 || mine[0] != want {
		t.Errorf("my crews = %+v, want just %+v", mine, want)
	}
}

// TestACodeBelongsToOneCrewAlone: two crews on one code would let a join land in the wrong one,
// so a taken code is ErrDuplicate and the handler draws again.
func TestACodeBelongsToOneCrewAlone(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	aCrew(t, repo, maya, "Dorm 4B", "123456")

	_, err := repo.CreateCrew(t.Context(), ben, "Lab partners", "123456", crewCap)
	if !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("second crew on one code: err = %v, want ErrDuplicate", err)
	}
	// The refused crew left nothing behind, not even a membership.
	mine, err := repo.ListMyCrews(t.Context(), ben)
	if err != nil {
		t.Fatalf("list my crews: %v", err)
	}
	if len(mine) != 0 {
		t.Errorf("ben's crews = %+v, want none", mine)
	}
}

// TestJoiningByCodeCountsTheNewMemberOnce: a join adds the player, and joining again changes
// nothing, so a double tap cannot count anyone twice.
func TestJoiningByCodeCountsTheNewMemberOnce(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	made := aCrew(t, repo, maya, "Dorm 4B", "123456")

	for try := range 2 {
		joined, err := repo.JoinCrew(t.Context(), ben, "123456", crewCap)
		if err != nil {
			t.Fatalf("join %d: %v", try, err)
		}
		want := repository.Crew{CrewID: made.CrewID, Name: "Dorm 4B", Code: "123456", Members: 2}
		if joined != want {
			t.Errorf("join %d = %+v, want %+v", try, joined, want)
		}
	}
}

// TestAWrongCodeJoinsNothing: a code no crew holds must say so in words, not as a bare not found.
func TestAWrongCodeJoinsNothing(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	aCrew(t, repo, maya, "Dorm 4B", "123456")

	_, err := repo.JoinCrew(t.Context(), ben, "654321", crewCap)
	if !errors.Is(err, repository.ErrNoCrewForCode) {
		t.Errorf("err = %v, want ErrNoCrewForCode", err)
	}
}

// TestAPlayerIsInNoMoreCrewsThanTheCap: making or joining past the cap is refused, while joining
// a crew they are already in still answers, since it adds nothing.
func TestAPlayerIsInNoMoreCrewsThanTheCap(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	for i := range crewCap {
		aCrew(t, repo, maya, fmt.Sprintf("Crew %d", i), fmt.Sprintf("10000%d", i))
	}
	aCrew(t, repo, ben, "Ben's", "200000")

	_, err := repo.CreateCrew(t.Context(), maya, "One more", "300000", crewCap)
	if !errors.Is(err, repository.ErrTooManyCrews) {
		t.Errorf("make past the cap: err = %v, want ErrTooManyCrews", err)
	}
	_, err = repo.JoinCrew(t.Context(), maya, "200000", crewCap)
	if !errors.Is(err, repository.ErrTooManyCrews) {
		t.Errorf("join past the cap: err = %v, want ErrTooManyCrews", err)
	}
	_, err = repo.JoinCrew(t.Context(), maya, "100000", crewCap)
	if err != nil {
		t.Errorf("rejoin a crew already in, at the cap: %v", err)
	}
}

// TestJoinsAtOnceStayUnderTheCap: twenty joins at the same moment must not all slip under the cap
// before any of them counts.
func TestJoinsAtOnceStayUnderTheCap(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	owner := aCrewMaker(t, repo, "test_owner")
	const crews = 20
	for i := range crews {
		// The owner is under no cap here: these crews only give maya somewhere to join.
		_, err := repo.CreateCrew(t.Context(), owner, fmt.Sprintf("Crew %d", i), fmt.Sprintf("%06d", 500000+i), crews)
		if err != nil {
			t.Fatalf("create crew %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	var joined, refused atomic.Int32
	for i := range crews {
		wg.Go(func() {
			_, err := repo.JoinCrew(t.Context(), maya, fmt.Sprintf("%06d", 500000+i), crewCap)
			switch {
			case err == nil:
				joined.Add(1)
			case errors.Is(err, repository.ErrTooManyCrews):
				refused.Add(1)
			default:
				t.Errorf("join %d: %v", i, err)
			}
		})
	}
	wg.Wait()

	if joined.Load() != crewCap || refused.Load() != crews-crewCap {
		t.Errorf("joined %d, refused %d, want %d and %d", joined.Load(), refused.Load(), crewCap, crews-crewCap)
	}
}

// TestACrewReadsAsMissingToANonMember: another crew must answer exactly like one that never was,
// so a crew id is no way in.
func TestACrewReadsAsMissingToANonMember(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	eve := aCrewMaker(t, repo, "test_eve")
	made := aCrew(t, repo, maya, "Dorm 4B", "123456")

	_, err := repo.CrewByID(t.Context(), eve, made.CrewID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("a non-member's read: err = %v, want ErrNotFound", err)
	}
	_, err = repo.CrewByID(t.Context(), maya, made.CrewID+1000)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("a crew that never was: err = %v, want ErrNotFound", err)
	}
}

// TestLeavingKeepsTheCrewForTheRestAndTheLastOneOutEndsIt: one member leaving must not end the
// crew for the others, and the last leaving must free its code.
func TestLeavingKeepsTheCrewForTheRestAndTheLastOneOutEndsIt(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	made := aCrew(t, repo, maya, "Dorm 4B", "123456")
	_, err := repo.JoinCrew(t.Context(), ben, "123456", crewCap)
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	// 1. The owner leaves; the crew is ben's alone now
	left, err := repo.LeaveCrew(t.Context(), maya, made.CrewID)
	if err != nil || !left {
		t.Fatalf("maya leaves: left = %v, err = %v", left, err)
	}
	still, err := repo.CrewByID(t.Context(), ben, made.CrewID)
	if err != nil {
		t.Fatalf("ben's crew after maya left: %v", err)
	}
	if still.Members != 1 {
		t.Errorf("members after maya left = %d, want 1", still.Members)
	}

	// 2. Leaving again changes nothing
	left, err = repo.LeaveCrew(t.Context(), maya, made.CrewID)
	if err != nil || left {
		t.Errorf("maya leaves again: left = %v, err = %v, want false and no error", left, err)
	}

	// 3. The last one out ends it, and its code joins nothing
	left, err = repo.LeaveCrew(t.Context(), ben, made.CrewID)
	if err != nil || !left {
		t.Fatalf("ben leaves: left = %v, err = %v", left, err)
	}
	_, err = repo.JoinCrew(t.Context(), maya, "123456", crewCap)
	if !errors.Is(err, repository.ErrNoCrewForCode) {
		t.Errorf("join an ended crew's code: err = %v, want ErrNoCrewForCode", err)
	}
}

// TestDeletingAnAccountTakesItsMembershipsAndItsCrews: a deleted player must leave no trace in a
// crew, and the crews they made go with them.
func TestDeletingAnAccountTakesItsMembershipsAndItsCrews(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	maya := aCrewMaker(t, repo, "test_maya")
	ben := aCrewMaker(t, repo, "test_ben")
	mayas := aCrew(t, repo, maya, "Maya's", "111111")
	bens := aCrew(t, repo, ben, "Ben's", "222222")
	_, err := repo.JoinCrew(t.Context(), ben, "111111", crewCap)
	if err != nil {
		t.Fatalf("ben joins maya's: %v", err)
	}
	_, err = repo.JoinCrew(t.Context(), maya, "222222", crewCap)
	if err != nil {
		t.Fatalf("maya joins ben's: %v", err)
	}

	deleted, err := repo.DeleteAccount(t.Context(), maya)
	if err != nil || !deleted {
		t.Fatalf("delete maya: deleted = %v, err = %v", deleted, err)
	}

	_, err = repo.CrewByID(t.Context(), ben, mayas.CrewID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("maya's crew after she was deleted: err = %v, want ErrNotFound", err)
	}
	left, err := repo.CrewByID(t.Context(), ben, bens.CrewID)
	if err != nil {
		t.Fatalf("ben's crew: %v", err)
	}
	if left.Members != 1 {
		t.Errorf("ben's crew members = %d, want 1 once maya is gone", left.Members)
	}
}
