package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/quasttyy/pr-reviewer/internal/repo"
)

// fakeUserStore — in-memory реализация UserStore для тестов UserService
type fakeUserStore struct {
	users                map[string]repo.UserRow
	errOnUpdateIsActive  error
	errOnGetByID         error
	lastUpdateUserID     string
	lastUpdateIsActive   bool
	updateCalled         bool
	getByIDCalledForUser string
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users: make(map[string]repo.UserRow),
	}
}

func (f *fakeUserStore) UpdateIsActive(ctx context.Context, userID string, isActive bool) (repo.UserRow, error) {
	f.updateCalled = true
	f.lastUpdateUserID = userID
	f.lastUpdateIsActive = isActive
	if f.errOnUpdateIsActive != nil {
		return repo.UserRow{}, f.errOnUpdateIsActive
	}
	row, ok := f.users[userID]
	if !ok {
		return repo.UserRow{}, errors.New("user not found")
	}
	row.IsActive = isActive
	f.users[userID] = row
	return row, nil
}

func (f *fakeUserStore) GetByID(ctx context.Context, userID string) (repo.UserRow, error) {
	f.getByIDCalledForUser = userID
	if f.errOnGetByID != nil {
		return repo.UserRow{}, f.errOnGetByID
	}
	row, ok := f.users[userID]
	if !ok {
		return repo.UserRow{}, errors.New("user not found")
	}
	return row, nil
}

// fakePRShortStore — in-memory реализация PRShortStore
type fakePRShortStore struct {
	prs             map[string][]repo.PRShortRow 
	errOnGetShort   error
	lastReviewerID  string
	getShortCalled  bool
}

func newFakePRShortStore() *fakePRShortStore {
	return &fakePRShortStore{
		prs: make(map[string][]repo.PRShortRow),
	}
}

func (f *fakePRShortStore) GetShortByReviewer(ctx context.Context, userID string) ([]repo.PRShortRow, error) {
	f.getShortCalled = true
	f.lastReviewerID = userID
	if f.errOnGetShort != nil {
		return nil, f.errOnGetShort
	}
	rows := f.prs[userID]
	return append([]repo.PRShortRow(nil), rows...), nil
}

func TestUserService_SetIsActiveAdmin_Success(t *testing.T) {
	users := newFakeUserStore()
	users.users["u1"] = repo.UserRow{
		UserID:   "u1",
		Username: "alice",
		TeamName: "backend",
		IsActive: false,
	}
	prs := newFakePRShortStore()
	svc := NewUserService(users, prs)

	row, err := svc.SetIsActiveAdmin(context.Background(), "u1", true)
	if err != nil {
		t.Fatalf("SetIsActiveAdmin() unexpected error: %v", err)
	}
	if !users.updateCalled {
		t.Fatalf("UpdateIsActive was not called")
	}
	if users.lastUpdateUserID != "u1" || users.lastUpdateIsActive != true {
		t.Fatalf("UpdateIsActive called with wrong args: userID=%q isActive=%v", users.lastUpdateUserID, users.lastUpdateIsActive)
	}
	if !row.IsActive {
		t.Fatalf("expected returned row.IsActive=true, got false")
	}
	if !users.users["u1"].IsActive {
		t.Fatalf("user state in store was not updated")
	}
}

func TestUserService_SetIsActiveAdmin_UserNotFound(t *testing.T) {
	users := newFakeUserStore()
	prs := newFakePRShortStore()
	svc := NewUserService(users, prs)

	_, err := svc.SetIsActiveAdmin(context.Background(), "missing", true)
	if err == nil {
		t.Fatalf("expected error for missing user, got nil")
	}
}

func TestUserService_SetIsActiveAdmin_UpdateError(t *testing.T) {
	users := newFakeUserStore()
	users.users["u1"] = repo.UserRow{UserID: "u1"}
	users.errOnUpdateIsActive = errors.New("update error")
	prs := newFakePRShortStore()
	svc := NewUserService(users, prs)

	_, err := svc.SetIsActiveAdmin(context.Background(), "u1", true)
	if err == nil || !errors.Is(err, users.errOnUpdateIsActive) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestUserService_GetUserReviews_Success(t *testing.T) {
	users := newFakeUserStore()
	users.users["u1"] = repo.UserRow{
		UserID:   "u1",
		Username: "alice",
		TeamName: "backend",
		IsActive: true,
	}
	prs := newFakePRShortStore()
	prs.prs["u1"] = []repo.PRShortRow{
		{ID: "pr1", Name: "Fix bug", AuthorID: "a1", Status: "OPEN"},
		{ID: "pr2", Name: "Add feature", AuthorID: "a2", Status: "MERGED"},
	}
	svc := NewUserService(users, prs)

	result, err := svc.GetUserReviews(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetUserReviews() unexpected error: %v", err)
	}
	if users.getByIDCalledForUser != "u1" {
		t.Fatalf("GetByID was not called with userID=u1, got %q", users.getByIDCalledForUser)
	}
	if !prs.getShortCalled || prs.lastReviewerID != "u1" {
		t.Fatalf("GetShortByReviewer was not called correctly, called=%v reviewer=%q", prs.getShortCalled, prs.lastReviewerID)
	}
	want := prs.prs["u1"]
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("GetUserReviews() mismatch.\n got: %+v\nwant: %+v", result, want)
	}
}

func TestUserService_GetUserReviews_UserNotFound(t *testing.T) {
	users := newFakeUserStore()
	prs := newFakePRShortStore()
	svc := NewUserService(users, prs)

	_, err := svc.GetUserReviews(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error for missing user, got nil")
	}
	if prs.getShortCalled {
		t.Fatalf("GetShortByReviewer should not be called when user does not exist")
	}
}

func TestUserService_GetUserReviews_GetByIDError(t *testing.T) {
	users := newFakeUserStore()
	users.errOnGetByID = errors.New("get error")
	prs := newFakePRShortStore()
	svc := NewUserService(users, prs)

	_, err := svc.GetUserReviews(context.Background(), "u1")
	if err == nil || !errors.Is(err, users.errOnGetByID) {
		t.Fatalf("expected getByID error, got %v", err)
	}
	if prs.getShortCalled {
		t.Fatalf("GetShortByReviewer should not be called when GetByID returns error")
	}
}

func TestUserService_GetUserReviews_GetShortError(t *testing.T) {
	users := newFakeUserStore()
	users.users["u1"] = repo.UserRow{UserID: "u1"}
	prs := newFakePRShortStore()
	prs.errOnGetShort = errors.New("short error")
	svc := NewUserService(users, prs)

	_, err := svc.GetUserReviews(context.Background(), "u1")
	if err == nil || !errors.Is(err, prs.errOnGetShort) {
		t.Fatalf("expected getShort error, got %v", err)
	}
}


