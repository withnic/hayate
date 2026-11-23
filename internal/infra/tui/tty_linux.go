package tui

import (
	"bufio"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

func ioctl(fd, request uintptr, mode *syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(mode)))
	if errno != 0 {
		return errno
	}
	return nil
}
func openTerminal() (*terminal, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	fd := f.Fd()
	var mode syscall.Termios
	if err = ioctl(fd, syscall.TCGETS, &mode); err != nil {
		f.Close()
		return nil, err
	}
	raw := mode
	raw.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG | syscall.IEXTEN
	raw.Iflag &^= syscall.IXON | syscall.ICRNL
	raw.Oflag &^= syscall.OPOST
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err = ioctl(fd, syscall.TCSETS, &raw); err != nil {
		f.Close()
		return nil, err
	}
	reader := bufio.NewReader(f)
	ready := func() bool {
		if reader.Buffered() > 0 {
			return true
		}
		var set syscall.FdSet
		if fd >= uintptr(len(set.Bits)*strconv.IntSize) {
			return false
		}
		set.Bits[fd/strconv.IntSize] |= 1 << (fd % strconv.IntSize)
		timeout := syscall.Timeval{Usec: 40000}
		count, _ := syscall.Select(int(fd)+1, &set, nil, nil, &timeout)
		return count > 0
	}
	return &terminal{
		Writer: f,
		close:  func() { _ = ioctl(fd, syscall.TCSETS, &mode); _ = f.Close() },
		readKey: func() (rune, error) {
			key, _, err := reader.ReadRune()
			if err != nil || key != 27 {
				return key, err
			}
			if !ready() {
				return keyCancel, nil
			}
			next, err := reader.ReadByte()
			if err != nil {
				return 0, err
			}
			if next != '[' && next != 'O' {
				return keyCancel, nil
			}
			if !ready() {
				return keyIgnore, nil
			}
			for {
				code, err := reader.ReadByte()
				if err != nil {
					return 0, err
				}
				switch code {
				case 'A':
					return keyUp, nil
				case 'B':
					return keyDown, nil
				}
				if code >= 0x40 || !ready() {
					return keyIgnore, nil
				}
			}
		},
	}, nil
}
