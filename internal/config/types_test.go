package config_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected time.Duration
		wantErr  bool
	}{
		{"Valid string duration", `"5s"`, 5 * time.Second, false},
		{"Valid string ms", `"100ms"`, 100 * time.Millisecond, false},
		{"Invalid format", `"abc"`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d config.Duration
			err := json.Unmarshal([]byte(tt.jsonStr), &d)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, d.ToDuration())
			}
		})
	}
}

func TestDuration_Set_Seconds(t *testing.T) {
	var d config.Duration
	err := d.Set("10")
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, d.ToDuration())
}

func TestDuration_Set_GoFormat(t *testing.T) {
	var d config.Duration
	err := d.Set("1m30s")
	require.NoError(t, err)
	assert.Equal(t, 90*time.Second, d.ToDuration())
}

func TestDuration_String(t *testing.T) {
	d := config.Duration(5 * time.Minute)
	assert.Equal(t, "5m0s", d.String())
}

func TestDuration_MarshalJSON(t *testing.T) {
	d := config.Duration(2 * time.Hour)
	data, err := json.Marshal(&d)
	require.NoError(t, err)
	assert.Equal(t, `"2h0m0s"`, string(data))
}
