package tui

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel           = syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode   = kernel.NewProc("SetConsoleMode")
	readConsoleInput = kernel.NewProc("ReadConsoleInputW")
)

func changeMode(f *os.File, mode uint32) error {
	result, _, err := setConsoleMode.Call(f.Fd(), uintptr(mode))
	if result == 0 {
		return err
	}
	return nil
}

func openTerminal() (*terminal, error) {
	input, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open console input: %w", err)
	}
	output, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		input.Close()
		return nil, fmt.Errorf("open console output: %w", err)
	}
	closeFiles := func() { input.Close(); output.Close() }
	var inMode, outMode uint32
	if err = syscall.GetConsoleMode(syscall.Handle(input.Fd()), &inMode); err != nil {
		closeFiles()
		return nil, fmt.Errorf("get console input mode: %w", err)
	}
	if err = syscall.GetConsoleMode(syscall.Handle(output.Fd()), &outMode); err != nil {
		closeFiles()
		return nil, fmt.Errorf("get console output mode: %w", err)
	}
	const processedInput = 0x0001
	const virtualTerminalOutput = 0x0004
	if err = changeMode(output, outMode|virtualTerminalOutput); err != nil {
		closeFiles()
		return nil, fmt.Errorf("enable virtual terminal output: %w", err)
	}
	if err = changeMode(input, inMode&^processedInput); err != nil {
		_ = changeMode(output, outMode)
		closeFiles()
		return nil, fmt.Errorf("set console input mode: %w", err)
	}
	var high rune
	var repeated rune
	var repeats uint16
	return &terminal{
		Writer: output,
		close:  func() { _ = changeMode(input, inMode); _ = changeMode(output, outMode); closeFiles() },
		readKey: func() (rune, error) {
			if repeats > 0 {
				repeats--
				return repeated, nil
			}
			for {
				// INPUT_RECORD contains a 4-byte header and a 16-byte event union.
				var record [5]uint32
				var count uint32
				result, _, err := readConsoleInput.Call(input.Fd(), uintptr(unsafe.Pointer(&record)), 1, uintptr(unsafe.Pointer(&count)))
				if result == 0 {
					return 0, err
				}
				event := (*[20]byte)(unsafe.Pointer(&record))
				if count == 0 || binary.LittleEndian.Uint16(event[0:2]) != 1 || record[1] == 0 {
					continue
				}
				key := rune(binary.LittleEndian.Uint16(event[14:16]))
				switch binary.LittleEndian.Uint16(event[10:12]) {
				case 0x26:
					key = keyUp
				case 0x28:
					key = keyDown
				case 0x1b:
					key = keyCancel
				}
				if 0xd800 <= key && key <= 0xdbff {
					high = key
					continue
				}
				if high != 0 {
					key = utf16.DecodeRune(high, key)
					high = 0
				}
				if key == 0 {
					continue
				}
				repeated = key
				if n := binary.LittleEndian.Uint16(event[8:10]); n > 0 {
					repeats = n - 1
				}
				return key, nil
			}
		},
	}, nil
}
