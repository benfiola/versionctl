package versionctl

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestVersionChangeCompare(t *testing.T) {
	t.Run("gt", func(t *testing.T) {
		require := require.New(t)
		l := VersionChange{Value: "major"}
		r := VersionChange{Value: "none"}

		d := l.Compare(r)

		require.Greater(d, 0)
	})

	t.Run("eq", func(t *testing.T) {
		require := require.New(t)
		l := VersionChange{Value: "none"}
		r := VersionChange{Value: "none"}

		d := l.Compare(r)

		require.Equal(0, d)
	})

	t.Run("lt", func(t *testing.T) {
		require := require.New(t)
		l := VersionChange{Value: "none"}
		r := VersionChange{Value: "major"}

		d := l.Compare(r)

		require.Less(d, 0)
	})
}
func TestVersionBump(t *testing.T) {
	t.Run("major", func(t *testing.T) {
		require := require.New(t)
		v := Version{Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}
		c := VersionChange{Value: "major"}

		nv := v.Bump(c)

		require.Equal(1, nv.Major)
		require.Equal(0, nv.Minor)
		require.Equal(0, nv.Patch)
		require.Equal(Prerelease{}, nv.Prerelease)
		require.Equal("", nv.Metadata)
	})

	t.Run("minor", func(t *testing.T) {
		require := require.New(t)
		v := Version{Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}
		c := VersionChange{Value: "minor"}

		nv := v.Bump(c)

		require.Equal(0, nv.Major)
		require.Equal(1, nv.Minor)
		require.Equal(0, nv.Patch)
		require.Equal(Prerelease{}, nv.Prerelease)
		require.Equal("", nv.Metadata)
	})

	t.Run("patch", func(t *testing.T) {
		require := require.New(t)
		v := Version{Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}
		c := VersionChange{Value: "patch"}

		nv := v.Bump(c)

		require.Equal(0, nv.Major)
		require.Equal(0, nv.Minor)
		require.Equal(1, nv.Patch)
		require.Equal(Prerelease{}, nv.Prerelease)
		require.Equal("", nv.Metadata)
	})

	t.Run("prerelease (token match)", func(t *testing.T) {
		require := require.New(t)
		v := Version{Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}
		c := VersionChange{Value: "prerelease", PrereleaseToken: "rc"}

		nv := v.Bump(c)

		require.Equal(0, nv.Major)
		require.Equal(0, nv.Minor)
		require.Equal(0, nv.Patch)
		require.Equal(Prerelease{Token: "rc", Count: 2}, nv.Prerelease)
		require.Equal("", nv.Metadata)
	})

	t.Run("prerelease (token mismatch)", func(t *testing.T) {
		require := require.New(t)
		v := Version{Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}
		c := VersionChange{Value: "prerelease", PrereleaseToken: "abc"}

		nv := v.Bump(c)

		require.Equal(0, nv.Major)
		require.Equal(0, nv.Minor)
		require.Equal(0, nv.Patch)
		require.Equal(Prerelease{Token: "abc", Count: 1}, nv.Prerelease)
		require.Equal("", nv.Metadata)
	})
}

func TestVersionCompare(t *testing.T) {
	t.Run("gt", func(t *testing.T) {
		require := require.New(t)
		l := Version{Major: 1}
		r := Version{}

		d := l.Compare(r)

		require.Greater(d, 0)
	})

	t.Run("eq", func(t *testing.T) {
		require := require.New(t)
		l := Version{}
		r := Version{}

		d := l.Compare(r)

		require.Equal(0, d)
	})

	t.Run("lt", func(t *testing.T) {
		require := require.New(t)
		l := Version{}
		r := Version{Major: 1}

		d := l.Compare(r)

		require.Less(d, 0)
	})

	t.Run("prerelease lt release", func(t *testing.T) {
		require := require.New(t)
		l := Version{Prerelease: Prerelease{Token: "rc", Count: 1}}
		r := Version{}

		d := l.Compare(r)

		require.Less(d, 0)
	})
}

func TestVersionDiff(t *testing.T) {
	t.Run("major", func(t *testing.T) {
		require := require.New(t)
		l := Version{Major: 1}
		r := Version{}

		vc := l.Diff(r)

		require.Equal("major", vc.Value)
	})

	t.Run("minor", func(t *testing.T) {
		require := require.New(t)
		l := Version{Minor: 1}
		r := Version{}

		vc := l.Diff(r)

		require.Equal("minor", vc.Value)
	})

	t.Run("patch", func(t *testing.T) {
		require := require.New(t)
		l := Version{Patch: 1}
		r := Version{}

		vc := l.Diff(r)

		require.Equal("patch", vc.Value)
	})
}

func TestVersionRelease(t *testing.T) {
	t.Run("major", func(t *testing.T) {
		require := require.New(t)
		v := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}

		r := v.Release()

		require.Equal(1, r.Major)
		require.Equal(2, r.Minor)
		require.Equal(3, r.Patch)
		require.Equal(Prerelease{}, r.Prerelease)
		require.Equal("", r.Metadata)
	})
}

func TestVersionString(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: Prerelease{Token: "rc", Count: 1}, Metadata: "metadata"}

	t.Run("docker", func(t *testing.T) {
		require := require.New(t)
		require.Equal("1.2.3-rc.1-metadata", v.String("docker"))
	})

	t.Run("git", func(t *testing.T) {
		require := require.New(t)
		require.Equal("v1.2.3-rc.1+metadata", v.String("git"))
	})

	t.Run("node", func(t *testing.T) {
		require := require.New(t)
		require.Equal("1.2.3-rc.1-metadata", v.String("node"))
	})

	t.Run("semver", func(t *testing.T) {
		require := require.New(t)
		require.Equal("1.2.3-rc.1+metadata", v.String("semver"))
		require.Equal("1.2.3-rc.1+metadata", v.String(""))
	})
}

func TestNewVersion(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		require := require.New(t)

		_, err := NewVersion("abcd")

		require.NotNil(err)
	})

	t.Run("release", func(t *testing.T) {
		require := require.New(t)

		v, err := NewVersion("1.2.3")

		require.Nil(err)
		require.Equal(1, v.Major)
		require.Equal(2, v.Minor)
		require.Equal(3, v.Patch)
		require.Equal(Prerelease{}, v.Prerelease)
		require.Equal("", v.Metadata)
	})

	t.Run("patch", func(t *testing.T) {
		require := require.New(t)

		v, err := NewVersion("1.2.3-rc.1")

		require.Nil(err)
		require.Equal(1, v.Major)
		require.Equal(2, v.Minor)
		require.Equal(3, v.Patch)
		require.Equal(Prerelease{Token: "rc", Count: 1}, v.Prerelease)
		require.Equal("", v.Metadata)
	})

	t.Run("metadata", func(t *testing.T) {
		require := require.New(t)

		v, err := NewVersion("1.2.3+metadata")

		require.Nil(err)
		require.Equal(1, v.Major)
		require.Equal(2, v.Minor)
		require.Equal(3, v.Patch)
		require.Equal(Prerelease{}, v.Prerelease)
		require.Equal("metadata", v.Metadata)
	})

	t.Run("patch + metadata", func(t *testing.T) {
		require := require.New(t)

		v, err := NewVersion("1.2.3-rc.1+metadata")

		require.Nil(err)
		require.Equal(1, v.Major)
		require.Equal(2, v.Minor)
		require.Equal(3, v.Patch)
		require.Equal(Prerelease{Token: "rc", Count: 1}, v.Prerelease)
		require.Equal("metadata", v.Metadata)
	})
}

func TestSetVersion(t *testing.T) {
	t.Run("sets pyproject.toml", func(t *testing.T) {
		require := require.New(t)
		d := t.TempDir()
		f := path.Join(d, "pyproject.toml")
		m := map[string]any{
			"project": map[string]any{
				"version": "0.0.0",
			},
		}
		b, err := toml.Marshal(m)
		require.Nil(err)
		err = os.WriteFile(f, b, 0o755)
		require.Nil(err)

		err = SetVersion("1.0.0", f)

		require.Nil(err)
		b, err = os.ReadFile(f)
		require.Nil(err)
		err = toml.Unmarshal(b, &m)
		require.Nil(err)
		require.Equal("1.0.0", m["project"].(map[string]any)["version"])
	})

	t.Run("sets package.json", func(t *testing.T) {
		require := require.New(t)
		d := t.TempDir()
		f := path.Join(d, "package.json")
		m := map[string]any{
			"version": "0.0.0",
		}
		b, err := json.Marshal(m)
		require.Nil(err)
		err = os.WriteFile(f, b, 0o755)
		require.Nil(err)

		err = SetVersion("1.0.0", f)

		require.Nil(err)
		b, err = os.ReadFile(f)
		require.Nil(err)
		err = json.Unmarshal(b, &m)
		require.Nil(err)
		require.Equal("1.0.0", m["version"])
	})

	t.Run("fails for unknown file type", func(t *testing.T) {
		require := require.New(t)
		d := t.TempDir()
		f := path.Join(d, "unknown.txt")
		_, err := os.Create(f)
		require.Nil(err)

		err = SetVersion("0.0.0", f)

		require.ErrorContains(err, "unknown file")
	})
}
