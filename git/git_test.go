package git

import (
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitClone(t *testing.T) {
	t.Parallel()
	g := New()

	var tt = []struct {
		repo        string
		tag         string
		expectedZip string
		expectedMod string
		err         error
	}{
		{
			repo:        "github.com/tmwalaszek/mod",
			tag:         "v1.0.0",
			expectedZip: "504b030414000800080000000000000000000000000000000000270000006769746875622e636f6d2f746d77616c61737a656b2f6d6f644076312e302e302f676f2e6d6f64cacd4f29cd495548cf2cc9284dd24bcecfd52fc92d4fcc492cae4acdd6cfcd4fe1e24acf5730d43332d533e002040000ffff504b0708e82eb3cd320000002c000000504b030414000800080000000000000000000000000000000000270000006769746875622e636f6d2f746d77616c61737a656b2f6d6f644076312e302e302f6d6f642e676f2a484cce4e4c4f55c8cd4fe102040000ffff504b0708b429088a120000000c000000504b0102140014000800080000000000e82eb3cd320000002c0000002700000000000000000000000000000000006769746875622e636f6d2f746d77616c61737a656b2f6d6f644076312e302e302f676f2e6d6f64504b0102140014000800080000000000b429088a120000000c0000002700000000000000000000000000870000006769746875622e636f6d2f746d77616c61737a656b2f6d6f644076312e302e302f6d6f642e676f504b05060000000002000200aa000000ee0000000000",
			expectedMod: "6d6f64756c65206769746875622e636f6d2f746d77616c61737a656b2f6d6f640a0a676f20312e32352e300a",
		},
		{
			repo: "github.com/tmwalaszek/mod2",
			err:  errors.New("failed to clone: exit status 128\nremote: Repository not found.\nfatal: repository 'https://github.com/tmwalaszek/mod2/' not found\n"),
		},
	}

	// todo(tmw) - make this errors more specific
	for _, tc := range tt {
		t.Run(fmt.Sprintf("repo %s tag %s", tc.repo, tc.tag), func(t *testing.T) {
			m, err := g.Clone(tc.repo, tc.tag)
			if tc.err != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.expectedZip, hex.EncodeToString(m.Zip))
			require.Equal(t, tc.expectedMod, hex.EncodeToString(m.Mod))

		})
	}
}

func TestGitFetchTags(t *testing.T) {
	t.Parallel()
	g := New()

	var tt = []struct {
		repo         string
		expectedTags []string
		err          error
	}{
		{
			repo:         "github.com/tmwalaszek/mod",
			expectedTags: []string{"v1.0.0"},
		},
		{
			repo:         "github.com/tmwalaszek/mod2",
			expectedTags: []string{},
			err:          errors.New("failed to fetch tags: exit status 128\nremote: Repository not found.\nfatal: repository 'https://github.com/tmwalaszek/mod2/' not found\n"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.repo, func(t *testing.T) {
			tags, err := g.FetchTags(tc.repo)
			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedTags, tags)
		})
	}
}
