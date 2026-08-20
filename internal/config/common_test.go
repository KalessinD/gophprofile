package config_test

import (
	"testing"

	"github.com/KalessinD/gophprofile/internal/config"
)

func TestParseConfigPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "empty args",
			args: []string{},
			want: "",
		},
		{
			name: "no config flag, other flags",
			args: []string{"-a=localhost:8080", "-r", "-k=secret"},
			want: "",
		},
		{
			name: "standalone -c with value",
			args: []string{"-a=localhost:8080", "-c", "config.json", "-r"},
			want: "config.json",
		},
		{
			name: "standalone -c without value at the end",
			args: []string{"-a=localhost:8080", "-c"},
			want: "",
		},
		{
			name: "standalone -config with value",
			args: []string{"-config", "/etc/app/config.yaml"},
			want: "/etc/app/config.yaml",
		},
		{
			name: "standalone -config without value",
			args: []string{"-config"},
			want: "",
		},
		{
			name: "equals -c=path",
			args: []string{"-a=:8080", "-c=./local/config.json"},
			want: "./local/config.json",
		},
		{
			name: "equals -c= (empty value)",
			args: []string{"-c="},
			want: "",
		},
		{
			name: "equals -config=path",
			args: []string{"-config=/tmp/cfg.json"},
			want: "/tmp/cfg.json",
		},
		{
			name: "equals -config= (empty value)",
			args: []string{"-config="},
			want: "",
		},
		{
			name: "first match wins (multiple config flags)",
			args: []string{"-c=first.json", "-config=second.json"},
			want: "first.json",
		},
		{
			name: "case sensitive check (uppercase C should not match)",
			args: []string{"-C=config.json"},
			want: "",
		},
		{
			name: "should not match -crypto-key as -c",
			args: []string{"-crypto-key=key.pem"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ParseConfigPath(tt.args)
			if got != tt.want {
				t.Errorf("parseConfigPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
