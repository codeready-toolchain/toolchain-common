package test

import (
	"os"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsetEnvVarAndRestore(t *testing.T) {
	t.Run("check unsetting and restoring of previously unset variable", func(t *testing.T) {
		// given
		varName := unsetVariable(t)

		// when
		resetFn := UnsetEnvVarAndRestore(t, varName)

		// then
		_, present := os.LookupEnv(varName)
		require.False(t, present)

		// finally
		resetFn()
		_, present = os.LookupEnv(varName)
		require.False(t, present)
	})

	t.Run("check unsetting and restoring of previously set variable", func(t *testing.T) {
		// given
		varName, val := setVariable(t)

		// when
		resetFn := UnsetEnvVarAndRestore(t, varName)

		// then
		_, present := os.LookupEnv(varName)
		require.False(t, present)

		// finally
		resetFn()
		valAfterRestoring, present := os.LookupEnv(varName)
		require.True(t, present)
		require.Equal(t, val, valAfterRestoring)
	})

	t.Run("check setting and restoring of previously set variable", func(t *testing.T) {
		// given
		varName := unsetVariable(t)

		// when
		resetFn := SetEnvVarAndRestore(t, varName, "newValue")

		// then
		newValue, present := os.LookupEnv(varName)
		require.True(t, present)
		assert.Equal(t, "newValue", newValue)

		// finally
		resetFn()
		_, present = os.LookupEnv(varName)
		require.False(t, present)
	})

	t.Run("check setting and restoring of previously unset variable", func(t *testing.T) {
		// given
		varName, val := setVariable(t)

		// when
		resetFn := SetEnvVarAndRestore(t, varName, "newValue")

		// then
		newValue, present := os.LookupEnv(varName)
		require.True(t, present)
		assert.Equal(t, "newValue", newValue)

		// finally
		resetFn()
		valAfterRestoring, present := os.LookupEnv(varName)
		require.True(t, present)
		require.Equal(t, val, valAfterRestoring)
	})

	t.Run("check setting and restoring of previously set variables", func(t *testing.T) {
		// given
		varName1, val1 := setVariable(t)
		varName2, val2 := setVariable(t)

		// when
		resetFn := SetEnvVarsAndRestore(t, Env(varName1, "newValue1"), Env(varName2, "newValue2"))

		// then
		newValue1, present := os.LookupEnv(varName1)
		require.True(t, present)
		assert.Equal(t, "newValue1", newValue1)

		newValue2, present := os.LookupEnv(varName2)
		require.True(t, present)
		assert.Equal(t, "newValue2", newValue2)

		// finally
		resetFn()
		valAfterRestoring1, present := os.LookupEnv(varName1)
		require.True(t, present)
		require.Equal(t, val1, valAfterRestoring1)

		valAfterRestoring2, present := os.LookupEnv(varName2)
		require.True(t, present)
		require.Equal(t, val2, valAfterRestoring2)
	})

	t.Run("check setting and restoring of previously unset variables", func(t *testing.T) {
		// given
		varName1 := unsetVariable(t)
		varName2 := unsetVariable(t)

		// when
		resetFn := SetEnvVarsAndRestore(t, Env(varName1, "newValue1"), Env(varName2, "newValue2"))

		// then
		newValue1, present := os.LookupEnv(varName1)
		require.True(t, present)
		assert.Equal(t, "newValue1", newValue1)

		newValue2, present := os.LookupEnv(varName2)
		require.True(t, present)
		assert.Equal(t, "newValue2", newValue2)

		// finally
		resetFn()
		_, present1 := os.LookupEnv(varName1)
		require.False(t, present1)

		_, present2 := os.LookupEnv(varName2)
		require.False(t, present2)
	})
}

func unsetVariable(t *testing.T) string {
	u, err := uuid.NewV4()
	require.NoError(t, err)
	varName := u.String()
	err = os.Unsetenv(varName)
	require.NoError(t, err)
	_, present := os.LookupEnv(varName)
	require.False(t, present)
	return varName
}

func setVariable(t *testing.T) (string, string) {
	u, err := uuid.NewV4()
	require.NoError(t, err)
	key := u.String()

	u, err = uuid.NewV4()
	require.NoError(t, err)
	value := u.String()

	err = os.Setenv(key, value)
	require.NoError(t, err)
	_, present := os.LookupEnv(key)
	require.True(t, present)
	return key, value
}
