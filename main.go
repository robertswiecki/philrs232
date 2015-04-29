/*
	Linux SICP (RS232C) protocol implementation for Philips BDM4065UC Display

	Author: Robert Swiecki (robert@swiecki.net)

	License: GNU GPL v3
*/

package main

import "flag"
import "fmt"
import "log"
import "os"
import "runtime"
import "sort"
import "syscall"
import "unsafe"

type Termios struct {
	Iflag uint32
	Oflag uint32
	Cflag uint32
	Lflag uint32
	Cc    [20]byte
}

const (
	TCFLSH  = 0x540B
	CRTSCTS = 0x80000000
	CBAUD   = 0x100F
)

func csum(v string) byte {
	var ret byte = 0
	for n := range v {
		ret ^= v[n]
	}
	return ret
}

func printhex(v []byte, max int) {
	for n := range v {
		if n >= max {
			break
		}
		fmt.Printf("%02x ", v[n])
	}
	fmt.Printf("\n")
}

func main() {
	runtime.LockOSThread()

	portFlag := flag.String("port", "/dev/ttyUSB0", "RS232C port")
	cmdFlag := flag.String("cmd", "", "Command")
	helpFlag := flag.Bool("help", false, "Help")
	speedFlag := flag.Int("speed", 9600, "ttyS speed")
	flag.Parse()

	commands := map[string]string{
		"ON":         "\xA6\x01\x00\x00\x00\x04\x01\x18\x02",
		"OFF":        "\xA6\x01\x00\x00\x00\x04\x01\x18\x01",
		"PIP-OFF":    "\xA6\x01\x00\x00\x00\x07\x01\x3C\x00\x00\x00\x00",
		"PIP-BL":     "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x00\x00\x00",
		"PIP-TL":     "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x01\x00\x00",
		"PIP-TR":     "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x02\x00\x00",
		"PIP-BR":     "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x03\x00\x00",
		"VOL0":       "\xA6\x01\x00\x00\x00\x04\x01\x44\x00",
		"VOL10":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x0A",
		"VOL20":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x14",
		"VOL30":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x1e",
		"VOL40":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x28",
		"VOL50":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x32",
		"VOL60":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x3C",
		"VOL70":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x46",
		"VOL80":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x50",
		"VOL90":      "\xA6\x01\x00\x00\x00\x04\x01\x44\x5A",
		"VOL100":     "\xA6\x01\x00\x00\x00\x04\x01\x44\x64",
		"PIC-NORM":   "\xA6\x01\x00\x00\x00\x04\x01\x3A\x00",
		"PIC-CUST":   "\xA6\x01\x00\x00\x00\x04\x01\x3A\x01",
		"PIC-REAL":   "\xA6\x01\x00\x00\x00\x04\x01\x3A\x02",
		"PIC-FULL":   "\xA6\x01\x00\x00\x00\x04\x01\x3A\x03",
		"PIC-219":    "\xA6\x01\x00\x00\x00\x04\x01\x3A\x04",
		"PIC-DYN  ":  "\xA6\x01\x00\x00\x00\x04\x01\x3A\x05",
		"MODE-GAME":  "\xA6\x01\x00\x00\x00\x0A\x01\x32\x64\x64\x64\x5A\x64\x00\x03",
		"MODE-OFF":   "\xA6\x01\x00\x00\x00\x0A\x01\x32\x5F\x32\x32\x32\x32\x00\x03",
		"TEMP-USER":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x00",
		"TEMP-NATU":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x01",
		"TEMP-3000":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x0D",
		"TEMP-6500":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x06",
		"TEMP-10000": "\xA6\x01\x00\x00\x00\x04\x01\x34\x09",
	}

	val, ok := commands[*cmdFlag]
	file, err := os.OpenFile(*portFlag, os.O_RDWR, 0666)
	if *helpFlag || !ok {
		fmt.Fprintf(os.Stderr, "Cmd: %s\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Commands: \n")

		var keys []string
		for k := range commands {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "  %s\n", k)
		}
		os.Exit(0)
	}

	if err != nil {
		log.Fatal(err)
	}

	_, _, errnop := syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(TCFLSH), uintptr(syscall.TCIOFLUSH))
	if errnop != 0 {
		log.Fatal(errnop)
	}

	var termios Termios
	_, _, errnop = syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)))
	if errnop != 0 {
		log.Fatal(errnop)
	}

	termios.Cflag &= ^uint32(CBAUD)
	termios.Cflag &= ^uint32(CRTSCTS)
	termios.Cflag &= ^uint32(syscall.PARENB)
	termios.Cflag &= ^uint32(syscall.CSTOPB)
	termios.Cflag |= syscall.CS8
	termios.Cflag |= syscall.CLOCAL
	termios.Cflag |= syscall.CREAD

	switch *speedFlag {
	case 1200:
		termios.Cflag |= syscall.B1200
	case 9600:
		termios.Cflag |= syscall.B9600
	case 19200:
		termios.Cflag |= syscall.B19200
	case 38400:
		termios.Cflag |= syscall.B38400
	case 57600:
		termios.Cflag |= syscall.B57600
	case 115200:
		termios.Cflag |= syscall.B115200
	default:
		log.Fatal("Unknown speed: ", *speedFlag)
	}

	_, _, errnop = syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&termios)))
	if errnop != 0 {
		log.Fatal(errnop)
	}

	cs := csum(val)
	request := append([]byte(val), cs)

	fmt.Printf("Request:  ")
	printhex(request, len(request))

	_, err = file.Write(request)
	if err != nil {
		log.Fatal("file.Write(ttyS): ", err)
	}

	resp := make([]byte, 32)
	n, err := file.Read(resp)
	if err != nil {
		log.Fatal("file.Read(ttyS): ", err)
	}

	fmt.Printf("Response: ")
	printhex(resp, n)
}
