package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-steputils/stepconf"
)

func Test_upgrade(t *testing.T) {
	type cfgs struct {
		UpdatePriority int `env:"update_priority,range[0..5]"`
		Input          string
		WantErr        bool
	}

	for _, cfg := range []cfgs{
		{
			Input:   "5",
			WantErr: false,
		},
		{
			Input:   "0",
			WantErr: false,
		},
		{
			Input:   "2",
			WantErr: false,
		},
		{
			Input:   "6",
			WantErr: true,
		},
		{
			Input:   "2000",
			WantErr: true,
		},
		{
			Input:   "4.9",
			WantErr: true,
		},
	} {
		if err := os.Setenv("update_priority", cfg.Input); err != nil {
			t.Fatal(err)
		}

		if err := stepconf.Parse(&cfg); err != nil && !cfg.WantErr {
			t.Fatal(err)
		}
	}
}

func Test_fraction(t *testing.T) {
	type cfgs struct {
		UserFraction float64 `env:"user_fraction,range]0.0..1.0["`
		Input        string
		Value        float64
		WantErr      bool
	}

	for _, cfg := range []cfgs{
		{
			Input:   "",
			Value:   0,
			WantErr: false,
		},
		{
			Input:   "0.3",
			Value:   0.3,
			WantErr: false,
		},
		{
			Input:   "0",
			Value:   0,
			WantErr: true,
		},
	} {
		if err := os.Setenv("user_fraction", cfg.Input); err != nil {
			t.Fatal(err)
		}

		if err := stepconf.Parse(&cfg); err != nil && !cfg.WantErr {
			t.Fatal(err)
		}

		if cfg.UserFraction != cfg.Value {
			t.Fatal("eeeh man")
		}
	}
}

func Test_parseAppList(t *testing.T) {
	tests := []struct {
		name     string
		list     string
		wantApps []string
	}{
		{
			name:     "empty app list",
			list:     "",
			wantApps: nil,
		},
		{
			name:     "newline separated list",
			list:     "app.apk\napp.aab\n \n",
			wantApps: []string{"app.apk", "app.aab"},
		},
		{
			name:     "pipe separated list",
			list:     "|app.apk|app.aab|",
			wantApps: []string{"app.apk", "app.aab"},
		},
		{
			name:     "pipe and newline separated list",
			list:     "\napp1.apk|app2.apk\napp.aab|",
			wantApps: []string{"app1.apk", "app2.apk", "app.aab"},
		},
		{
			name:     "pipe and newline separated list",
			list:     "/bitrise/deploy/app-bitrise-signed.aab\n/bitrise/deploy/app.aab",
			wantApps: []string{"/bitrise/deploy/app-bitrise-signed.aab", "/bitrise/deploy/app.aab"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotApps := parseAppList(tt.list); !reflect.DeepEqual(gotApps, tt.wantApps) {
				t.Errorf("parseAppList() = %v, want %v", gotApps, tt.wantApps)
			}
		})
	}
}

func TestConfigs_appPaths(t *testing.T) {
	tests := []struct {
		name         string
		config       Configs
		wantApps     []string
		wantWarnings []string
	}{
		{
			name: "empty test",
			config: Configs{
				AppPath: "",
			},
			wantApps:     nil,
			wantWarnings: nil,
		},
		{
			name: "prefers aab",
			config: Configs{
				AppPath: "app.apk|app.aab",
			},
			wantApps:     []string{"app.aab"},
			wantWarnings: []string{"Both .aab and .apk files provided, using the .aab file(s): app.aab"},
		},
		{
			name: "uses first aab",
			config: Configs{
				AppPath: "app.aab\napp1.aab",
			},
			wantApps:     []string{"app.aab"},
			wantWarnings: []string{"More than 1 .aab files provided, using the first: app.aab"},
		},
		{
			name: "unknown extension",
			config: Configs{
				AppPath: "mapping.txt",
			},
			wantApps:     nil,
			wantWarnings: []string{"unknown app path extension in path: mapping.txt, supported extensions: .apk, .aab"},
		},
		{
			name: "newline (\n) as a character",
			config: Configs{
				AppPath: `/bitrise/deploy/app-bitrise-signed.aab\n/bitrise/deploy/app.aab`,
			},
			wantApps:     []string{"/bitrise/deploy/app-bitrise-signed.aab"},
			wantWarnings: []string{"More than 1 .aab files provided, using the first: /bitrise/deploy/app-bitrise-signed.aab"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApps, gotWarnings := tt.config.appPaths()
			if !reflect.DeepEqual(gotApps, tt.wantApps) {
				t.Errorf("Configs.appPaths() gotApps = %v, want %v", gotApps, tt.wantApps)
			}
			if !reflect.DeepEqual(gotWarnings, tt.wantWarnings) {
				t.Errorf("Configs.appPaths() gotWarnings = %v, want %v", gotWarnings, tt.wantWarnings)
			}
		})
	}
}

func Test_expansionFiles(t *testing.T) {
	tests := []struct {
		name                    string
		appPaths                []string
		expansionFilePathConfig string
		entries                 []string
		wantErr                 bool
	}{
		{"mainOnly", []string{"x.apk", "y.apk", "z.apk"}, "main:a.obb|main:b.obb|main:c.obb", []string{"main:a.obb", "main:b.obb", "main:c.obb"}, false},
		{"pathOnly", []string{"x.apk", "y.apk", "z.apk"}, "patch:a.obb|patch:b.obb|patch:c.obb", []string{"patch:a.obb", "patch:b.obb", "patch:c.obb"}, false},
		{"mixed", []string{"x.apk", "y.apk", "z.apk"}, "main:a.obb|patch:b.obb|patch:c.obb", []string{"main:a.obb", "patch:b.obb", "patch:c.obb"}, false},
		{"omit", []string{"x.apk", "y.apk", "z.apk"}, "main:a.obb||patch:c.obb", []string{"main:a.obb", "", "patch:c.obb"}, false},
		{"multipleOmit", []string{"w.apk", "x.apk", "y.apk", "z.apk"}, "main:a.obb|||patch:c.obb", []string{"main:a.obb", "", "", "patch:c.obb"}, false},
		{"invalid1", []string{"x.apk", "y.apk", "z.apk"}, "main:a.obb", []string{}, true},
		{"invalid2", []string{"x.apk", "y.apk", "z.apk"}, "", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expansionFiles(tt.appPaths, tt.expansionFilePathConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("expansionFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.entries) {
				t.Errorf("expansionFiles() got1 = %v, want %v", got, tt.entries)
			}
		})
	}
}
