package osenv

import "testing"

func TestPathConversion(t *testing.T) {
	p, ok := WindowsToWSL("C:\\my work\\[id]")
	if !ok || p != "/mnt/c/my work/[id]" {
		t.Fatalf("mount: %q", p)
	}
	win, ok := WSLToWindows(p)
	if !ok || win != "C:\\my work\\[id]" {
		t.Fatalf("Windows: %q", win)
	}
	if got := WSLToUNC("Ubuntu", "/home/dev"); got != "\\\\wsl.localhost\\Ubuntu\\home\\dev" {
		t.Fatalf("UNC: %q", got)
	}
	if _, ok := WindowsToWSL("C:relative"); ok {
		t.Fatal("accepted drive-relative path")
	}
}
