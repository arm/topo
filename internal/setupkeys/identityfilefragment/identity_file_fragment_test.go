package identityfilefragment_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys/identityfilefragment"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestIdentityFileFragmentWrite(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("writes the identity file fragment to the target SSH config", func(t *testing.T) {
			tmp := t.TempDir()
			testutil.SetHomeDir(t, tmp)

			dest := ssh.NewDestination("user@example.com:2222")
			targetSlug := dest.Slugify()
			privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetSlug))

			op := identityfilefragment.NewIdentityFileFragmentWrite(privKeyPath, targetSlug, dest)

			var buf bytes.Buffer
			require.NoError(t, op.Run(&buf))
			require.Empty(t, buf.String())

			mainConfigPath := filepath.Join(tmp, ".ssh", "config")
			wantIncludedFragmentPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf"))
			testutil.AssertFileContents(t, fmt.Sprintf("Include %s\n\n", wantIncludedFragmentPath), mainConfigPath)

			fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetSlug))
			wantFragmentContents := fmt.Sprintf(`Host example.com
  HostName example.com
  User user
  Port 2222
  IdentityFile %s
  IdentitiesOnly yes
`, filepath.ToSlash(privKeyPath))
			testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
		})
	})
}
