package domain

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"slices"
)

type tokenScopesKey struct{}
type tokenWorkspacesKey struct{}
type effortSystemKey struct{}

var ErrEffortForbidden = errors.New("Efforts operation is forbidden")

// WithTokenScopes preserves the PAT ceiling below the HTTP layer. Even an empty
// scope list marks a token; absence of this context value denotes a session.
func WithTokenScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, tokenScopesKey{}, slices.Clone(scopes))
}

// Nil means unrestricted; a non-nil empty allowlist permits no workspace.
func WithTokenWorkspaces(ctx context.Context, slugs []string) context.Context {
	return context.WithValue(ctx, tokenWorkspacesKey{}, slices.Clone(slugs))
}

func TokenWorkspaces(ctx context.Context) []string {
	slugs, _ := ctx.Value(tokenWorkspacesKey{}).([]string)
	return slices.Clone(slugs)
}

func IsToken(ctx context.Context) bool {
	_, ok := ctx.Value(tokenScopesKey{}).([]string)
	return ok
}

func CheckEffortToken(ctx context.Context) error {
	if scopes, ok := ctx.Value(tokenScopesKey{}).([]string); ok {
		if !slices.Contains(scopes, PermEffortsRead) || !slices.Contains(scopes, PermEffortsWrite) {
			return ErrEffortForbidden
		}
	}
	return nil
}

// WithGitHubEffortActor is used only by the authenticated GitHub automation path.
// It does not grant private-team visibility or permission to create worklogs.
func WithGitHubEffortActor(ctx context.Context) context.Context {
	return context.WithValue(ctx, effortSystemKey{}, "github")
}

func EffortActorFromContext(ctx context.Context) (EffortActor, error) {
	if source, ok := ctx.Value(effortSystemKey{}).(string); ok {
		return EffortActor{Kind: "system", SystemSource: &source}, nil
	}
	access, ok := AccessFromContext(ctx)
	if !ok || access.UserID == uuid.Nil {
		return EffortActor{}, ErrEffortForbidden
	}
	return EffortActor{Kind: "user", UserID: &access.UserID}, nil
}

// Efforts projections are never available to public contexts or unscoped PATs.
func CanReadEfforts(ctx context.Context) bool {
	access, ok := AccessFromContext(ctx)
	if !ok || access.UserID == uuid.Nil {
		return false
	}
	scopes, token := ctx.Value(tokenScopesKey{}).([]string)
	return !token || slices.Contains(scopes, PermEffortsRead)
}
