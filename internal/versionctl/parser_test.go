package versionctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultParser(t *testing.T) {
	t.Run("no tag match", func(t *testing.T) {
		require := require.New(t)
		p, err := NewParser("default", &ParserOpts{})
		require.Nil(err)

		vc := p.Parse("")

		require.Equal("none", vc.Value)
	})

	t.Run("tag match", func(t *testing.T) {
		require := require.New(t)
		p, err := NewParser("default", &ParserOpts{
			Tags: map[string]string{
				"tag:": "minor",
			},
		})
		require.Nil(err)

		vc := p.Parse("tag: test")

		require.Equal("minor", vc.Value)
	})

	t.Run("breaking change match", func(t *testing.T) {
		require := require.New(t)
		p, err := NewParser("default", &ParserOpts{
			BreakingChangeTags: []string{"bct:"},
			Tags: map[string]string{
				"tag:": "minor",
			},
		})
		require.Nil(err)

		vc := p.Parse("tag: test\nbct: other")

		require.Equal("major", vc.Value)
	})
}
