package versionctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("no tag match", func(t *testing.T) {
		require := require.New(t)
		p := Parser{}

		vc := p.parse("")

		require.Equal("none", vc.Value)
	})

	t.Run("tag match", func(t *testing.T) {
		require := require.New(t)
		p := Parser{Tags: map[string]string{
			"tag:": "minor",
		}}

		vc := p.parse("tag: test")

		require.Equal("minor", vc.Value)
	})

	t.Run("breaking change match", func(t *testing.T) {
		require := require.New(t)
		p := Parser{
			BreakingChangeTags: []string{"bct:"},
			Tags: map[string]string{
				"tag:": "minor",
			},
		}

		vc := p.parse("tag: test\nbct: other")

		require.Equal("major", vc.Value)
	})
}
