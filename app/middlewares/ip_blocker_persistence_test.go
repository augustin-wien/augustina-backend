package middlewares

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent/blockedip"
	"github.com/stretchr/testify/require"
)

func TestIPBlockerPersistsBlocksAndStrikes(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	config.InitConfig()
	require.NoError(t, database.Db.InitEmptyTestDb())
	client := database.Db.EntClient
	ctx := context.Background()

	const blockedIP = "198.51.100.10"
	const strikeIP = "198.51.100.11"
	const expiredIP = "198.51.100.12"
	_, _ = client.BlockedIP.Delete().Where(blockedip.IPIn(blockedIP, strikeIP, expiredIP)).Exec(ctx)
	t.Cleanup(func() {
		_, _ = client.BlockedIP.Delete().Where(blockedip.IPIn(blockedIP, strikeIP, expiredIP)).Exec(ctx)
	})

	writer := &IPBlocker{blockedIPs: map[string]time.Time{}, strikes: map[string]int{}, client: client}
	expiry := time.Now().Add(time.Hour)

	// Writing twice exercises both the insert and the update path
	require.NoError(t, writer.saveStrikeToDB(ctx, strikeIP, 1))
	require.NoError(t, writer.saveStrikeToDB(ctx, strikeIP, 3))
	require.NoError(t, writer.saveStrikeToDB(ctx, blockedIP, 2))
	require.NoError(t, writer.saveBlockToDB(ctx, blockedIP, expiry))
	require.NoError(t, writer.saveBlockToDB(ctx, expiredIP, time.Now().Add(-time.Hour)))

	// A fresh blocker restores the state from the database
	orig := GlobalBlocker
	GlobalBlocker = &IPBlocker{blockedIPs: map[string]time.Time{}, strikes: map[string]int{}}
	t.Cleanup(func() { GlobalBlocker = orig })
	InitIPBlocker(client)

	require.True(t, GlobalBlocker.IsBlocked(blockedIP))
	require.False(t, GlobalBlocker.IsBlocked(strikeIP))
	require.False(t, GlobalBlocker.IsBlocked(expiredIP))
	require.Equal(t, 3, GlobalBlocker.strikes[strikeIP])
	_, hasStrikes := GlobalBlocker.strikes[blockedIP]
	require.False(t, hasStrikes, "a block resets the strikes")

	// With a client set, blocking and strikes are written in the background
	GlobalBlocker.BlockIP("198.51.100.13", time.Minute)
	GlobalBlocker.AddStrike("198.51.100.14")
	t.Cleanup(func() {
		_, _ = client.BlockedIP.Delete().Where(blockedip.IPIn("198.51.100.13", "198.51.100.14")).Exec(ctx)
	})
	require.Eventually(t, func() bool {
		n, err := client.BlockedIP.Query().Where(blockedip.IPIn("198.51.100.13", "198.51.100.14")).Count(ctx)
		return err == nil && n == 2
	}, 5*time.Second, 50*time.Millisecond)
}
