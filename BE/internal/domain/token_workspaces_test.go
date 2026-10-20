package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenWorkspacesPreservesCeiling(t *testing.T) {
	require.Nil(t, TokenWorkspaces(context.Background()))
	require.Nil(t, TokenWorkspaces(WithTokenWorkspaces(context.Background(), nil)))
	require.NotNil(t, TokenWorkspaces(WithTokenWorkspaces(context.Background(), []string{})))
	input := []string{"a"}
	ctx := WithTokenWorkspaces(context.Background(), input)
	input[0] = "b"
	require.Equal(t, []string{"a"}, TokenWorkspaces(ctx))
	output := TokenWorkspaces(ctx)
	output[0] = "c"
	require.Equal(t, []string{"a"}, TokenWorkspaces(ctx))
}
