// Author: Caden Lund
// Package account serves account.v1 AccountService: making the device account, saying whose
// session a request carries, saving the player's kitchen, and deleting the account for good.
package account

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// The public rpcs' limits per client IP. Device accounts are free to make, so creating one is
// capped tight enough to stop a script and loose enough for a classroom on one network. Checking
// a handle happens at every pause in typing, for everyone on that network.
const (
	createsPerIP = 30
	createWindow = 10 * time.Minute
	checksPerIP  = 120
	checkWindow  = time.Minute
)

// handleTaken goes under the handle input when the unique constraint refuses it.
const handleTaken = "that handle is taken"

// undoTimeout bounds deleting an account whose sign-in failed. The delete runs even after the
// client hangs up, so it needs a deadline of its own.
const undoTimeout = 5 * time.Second

// Account implements account.v1 AccountService.
type Account struct {
	repo   *repository.Repository
	auth   *auth.Auth
	gate   handlers.Gate
	limits Limits
	log    *slog.Logger
}

// Config is what Account needs to exist.
type Config struct {
	Repo *repository.Repository
	Auth *auth.Auth
	// Limits caps the public rpcs. main.go passes DefaultLimits; the zero value limits nothing.
	Limits Limits
	Log    *slog.Logger
}

// Limits holds one limiter per public rpc, each keyed by client IP. A nil one limits nothing,
// which is how test fixtures turn limits off.
type Limits struct {
	Create *handlers.Limiter
	Check  *handlers.Limiter
}

// DefaultLimits are the limits the API serves with.
func DefaultLimits() Limits {
	return Limits{
		Create: handlers.NewLimiter(createsPerIP, createWindow),
		Check:  handlers.NewLimiter(checksPerIP, checkWindow),
	}
}

// NewAccount builds the service. A nil Log discards.
func NewAccount(cfg Config) *Account {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Account{
		repo:   cfg.Repo,
		auth:   cfg.Auth,
		gate:   handlers.Gate{Auth: cfg.Auth},
		limits: cfg.Limits,
		log:    log,
	}
}

// fail is the one way an error leaves this service.
func (h *Account) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// undoCreate deletes the account CreateAccount just made, once its sign-in has failed. It outlives
// the request, so a client that hung up mid-sign-in still gets its handle back. A failed delete is
// logged here rather than returned: the caller answers with the sign-in's error, which is never
// logged when the client hung up.
func (h *Account) undoCreate(ctx context.Context, accountID int64) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), undoTimeout)
	defer cancel()
	_, err := h.repo.DeleteAccount(ctx, accountID)
	if err != nil {
		h.log.Warn("undo create account", "account_id", accountID, "err", err)
	}
}

// CreateAccount makes the device account and signs the session in.
func (h *Account) CreateAccount(
	ctx context.Context, req *connect.Request[accountv1.CreateAccountRequest],
) (*connect.Response[accountv1.CreateAccountResponse], error) {
	const op = "create account"

	// 1. Public by design: this is how a device gets its account. Limited per client IP instead.
	err := h.limits.Create.Allow(handlers.ClientIP(req.Peer()))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate both fields, so the form shows every problem at once
	handle, avatar := normalizeHandle(req.Msg.Handle), normalizeAvatar(req.Msg.Avatar)
	invalid := append(handlers.Validate(handleInput{handle}), handlers.Validate(avatarInput{avatar})...)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write. The handle is app.account's only unique constraint, so a duplicate means it
	// is taken, and that goes under the input
	acct, err := h.repo.CreateAccount(ctx, handle, avatar)
	if errors.Is(err, repository.ErrDuplicate) {
		return nil, h.fail(op, handlers.FieldErrors{{Field: "handle", Message: handleTaken}})
	}
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. Making the account is the sign-in. The session is the account's only way in, so if it
	// can't be saved the account goes too, and the handle is free for the next try
	err = h.auth.SignIn(ctx, acct.AccountID)
	if err != nil {
		h.undoCreate(ctx, acct.AccountID)
		return nil, h.fail(op, err)
	}

	account, err := toProtoAccount(acct)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&accountv1.CreateAccountResponse{Account: account}), nil
}

// CheckHandle says whether a handle is free.
func (h *Account) CheckHandle(
	ctx context.Context, req *connect.Request[accountv1.CheckHandleRequest],
) (*connect.Response[accountv1.CheckHandleResponse], error) {
	const op = "check handle"

	// 1. Public by design: onboarding asks before the account exists. Limited per client IP
	// instead.
	err := h.limits.Check.Allow(handlers.ClientIP(req.Peer()))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate, so a handle that could never be taken says why
	handle := normalizeHandle(req.Msg.Handle)
	invalid := handlers.Validate(handleInput{handle})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Look it up. Only an answer for now: CreateAccount's insert is the real check
	taken, err := h.repo.HandleTaken(ctx, handle)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&accountv1.CheckHandleResponse{Available: !taken}), nil
}

// UpdateKitchen replaces the caller's kitchen.
func (h *Account) UpdateKitchen(
	ctx context.Context, req *connect.Request[accountv1.UpdateKitchenRequest],
) (*connect.Response[accountv1.UpdateKitchenResponse], error) {
	const op = "update kitchen"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Every value back to its stored word, refusing each one outside its closed set
	edit, invalid := fromProtoKitchen(req.Msg.GetKitchen())
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write, on the caller's own row
	updated, err := h.repo.UpdateKitchen(ctx, me.AccountID, edit)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !updated {
		return nil, h.fail(op, repository.ErrNotFound)
	}

	// 4. Re-read, so the app sees what was stored
	stored, err := h.repo.AccountByID(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	account, err := toProtoAccount(stored)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&accountv1.UpdateKitchenResponse{Account: account}), nil
}

// WhoAmI reads the session's account.
func (h *Account) WhoAmI(
	ctx context.Context, _ *connect.Request[accountv1.WhoAmIRequest],
) (*connect.Response[accountv1.WhoAmIResponse], error) {
	const op = "who am i"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	account, err := toProtoAccount(me)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&accountv1.WhoAmIResponse{Account: account}), nil
}

// DeleteAccount deletes the caller's account and everything of theirs, then ends its sessions.
func (h *Account) DeleteAccount(
	ctx context.Context, _ *connect.Request[accountv1.DeleteAccountRequest],
) (*connect.Response[accountv1.DeleteAccountResponse], error) {
	const op = "delete account"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. The write: one statement, and every table that references the account cascades
	// (docs/contracts.md). Another call can delete it between the gate and here
	deleted, err := h.repo.DeleteAccount(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !deleted {
		return nil, h.fail(op, repository.ErrNotFound)
	}

	// 3. Its sessions. A session left behind already reads as signed out, because the gate
	// re-reads the account on every call, so a failure here is logged and never returned
	err = h.auth.EndSessions(ctx, me.AccountID)
	if err != nil {
		h.log.Warn("end sessions", "account_id", me.AccountID, "err", err)
	}
	return connect.NewResponse(&accountv1.DeleteAccountResponse{}), nil
}
