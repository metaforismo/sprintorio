package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceTransferPermissionIsLimitedToOwnerAndAdmin(t *testing.T) {
	require.True(t, HasPermission(RoleOwner, PermWorkspaceTransfer))
	require.True(t, HasPermission(RoleAdmin, PermWorkspaceTransfer))
	require.False(t, HasPermission(RoleMember, PermWorkspaceTransfer))
	require.False(t, HasPermission(RoleGuest, PermWorkspaceTransfer))
}
