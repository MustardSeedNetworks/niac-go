//go:build windows

package main

import "testing"

func TestRequireWithin(t *testing.T) {
	const root = `C:\Program Files\NiAC`
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "the binary", path: root + `\niac.exe`},
		{name: "nested", path: root + `\bin\niac.exe`},
		{name: "different case", path: `c:\program files\niac\niac.exe`},
		{name: "the root itself", path: root, wantErr: true},
		{name: "sibling with the same prefix", path: `C:\Program Files\NiACevil\niac.exe`, wantErr: true},
		{name: "parent", path: `C:\Program Files\niac.exe`, wantErr: true},
		{name: "dot-dot escape", path: root + `\..\Evil\niac.exe`, wantErr: true},
		{name: "user profile", path: `C:\Users\alice\Downloads\niac.exe`, wantErr: true},
		{name: "another drive", path: `D:\Program Files\NiAC\niac.exe`, wantErr: true},
		{name: "relative", path: `niac.exe`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireWithin(root, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("requireWithin(%q) = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
