package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	defaultEmptyMap = make(map[string]string)
)

func Test_genSchemaObjectID(t *testing.T) {
	t.Run("check none", func(t *testing.T) {
		result := genSchemaObjectID("", "sample", defaultEmptyMap)

		require.Equal(t, "sample", string(result))
	})
	t.Run("check single", func(t *testing.T) {
		result := genSchemaObjectID("sample", "sample", defaultEmptyMap)

		require.Equal(t, "sample.sample", string(result))
	})
	t.Run("check multiple", func(t *testing.T) {
		result := genSchemaObjectID("test.sample", "sample", defaultEmptyMap)

		require.Equal(t, "test.sample.sample", string(result))
	})
}

func Test_getAliasedTypeName(t *testing.T) {
	t.Run("check no change", func(t *testing.T) {
		result := getAliasedTypeName("mypackage.test", defaultEmptyMap)

		require.Equal(t, "mypackage.test", string(result))
	})

	t.Run("check empty rename", func(t *testing.T) {
		emptyMap := make(map[string]string)
		emptyString := ""
		emptyMap["mypackage"] = emptyString
		result := getAliasedTypeName("mypackage.test", emptyMap)

		require.Equal(t, "test", string(result))
	})

	t.Run("check type rename", func(t *testing.T) {
		emptyMap := make(map[string]string)
		emptyString := "newpackage"
		emptyMap["mypackage"] = emptyString
		result := getAliasedTypeName("mypackage.test", emptyMap)

		require.Equal(t, "newpackage.test", string(result))
	})
}

func Test_getAliasedPackageName(t *testing.T) {
	pkgName := "example/go/mypackage"
	t.Run("check no change", func(t *testing.T) {
		result := getAliasedPackageName(pkgName, defaultEmptyMap)

		require.Equal(t, "example/go/mypackage", string(result))
	})

	t.Run("check empty rename", func(t *testing.T) {
		emptyMap := make(map[string]string)
		emptyString := ""
		emptyMap["mypackage"] = emptyString
		result := getAliasedPackageName(pkgName, emptyMap)

		require.Equal(t, "", string(result))
	})

	t.Run("check rename alias package", func(t *testing.T) {
		emptyMap := make(map[string]string)
		emptyString := "newpackage"
		emptyMap["mypackage"] = emptyString
		result := getAliasedPackageName(pkgName, emptyMap)

		require.Equal(t, "newpackage", string(result))
	})
}
