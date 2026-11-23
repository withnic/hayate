package app

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/withnic/hayate/internal/domain"
)

func TestWindowsPowerShellWrapperRunsActionOnlyAfterMoving(t *testing.T) {
	var wrapper bytes.Buffer
	if err := Init(&wrapper, "powershell", "j"); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "app [id]'s")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "hayate-test-marker"), []byte("expected project"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(project, "missing")
	for _, path := range []string{project, missing} {
		command, err := domain.BuildCommand("powershell-legacy",
			domain.ProjectEntry{Env: domain.Windows, NativePath: path},
			domain.ActionConfig{Windows: "Write-Output 'action ran'"})
		if err != nil {
			t.Fatal(err)
		}
		if path == project {
			t.Setenv("HAYATE_TEST_SUCCESS", command)
		} else {
			t.Setenv("HAYATE_TEST_FAILURE", command)
		}
	}
	// Stand in for the interactive selector while exercising the generated
	// wrapper and evaluating its stdout in Windows PowerShell itself.
	script := wrapper.String() + `
$ErrorActionPreference = 'Stop'
function hayate.exe {
    if (($args -join ' ') -ne 'select --shell powershell-legacy') {
        throw "Unexpected arguments: $args"
    }
    $global:LASTEXITCODE = 0
    $env:HAYATE_TEST_COMMAND -split "\r?\n"
}
function Read-ProjectMarker {
    # Windows PowerShell can escape brackets in its provider location.
    # Inspect the working directory that a native action actually receives.
    $nativeLocation = cmd.exe /d /c cd
    return [IO.File]::ReadAllText([IO.Path]::Combine($nativeLocation, 'hayate-test-marker'))
}
$env:HAYATE_TEST_COMMAND = $env:HAYATE_TEST_SUCCESS
$result = j
if ($result -ne 'action ran') {
    throw "Expected the action to run, got: $result"
}
if ((Read-ProjectMarker) -ne 'expected project') {
    throw "Expected to enter the project, got: $(Get-Location)"
}
$env:HAYATE_TEST_COMMAND = $env:HAYATE_TEST_FAILURE
$ErrorActionPreference = 'SilentlyContinue'
$result = j
$ErrorActionPreference = 'Stop'
if ($null -ne $result) {
    throw "A failed move must not run the action, got: $result"
}
if ((Read-ProjectMarker) -ne 'expected project') {
    throw "A failed move must not change location, got: $(Get-Location)"
}
`
	encoded := make([]byte, 0, len(script)*2)
	for _, char := range utf16.Encode([]rune(script)) {
		encoded = binary.LittleEndian.AppendUint16(encoded, char)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Windows PowerShell: %v\n%s", err, out)
	}
}
