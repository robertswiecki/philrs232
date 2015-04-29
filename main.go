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
import "syscall"
import "unsafe"

type Termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]byte
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

	portFlag  := flag.String("port", "/dev/ttyUSB0", "RS232C port")
	cmdFlag   := flag.String("cmd", "", "Command")
	helpFlag  := flag.Bool("help", false, "Help")
	speedFlag := flag.Int("speed", 9600, "ttyS speed")
	flag.Parse()

	commands := map[string]string {
		"ON":      "\xA6\x01\x00\x00\x00\x04\x01\x18\x02",
		"OFF":     "\xA6\x01\x00\x00\x00\x04\x01\x18\x01",
		"PIPOFF":  "\xA6\x01\x00\x00\x00\x07\x01\x3C\x00\x00\x00\x00",
		"PIPBL":   "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x00\x00\x00",
		"PIPTL":   "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x01\x00\x00",
		"PIPTR":   "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x02\x00\x00",
		"PIPBR":   "\xA6\x01\x00\x00\x00\x07\x01\x3C\x01\x03\x00\x00",
		"PIPGET":  "\xA6\x01\x00\x00\x00\x03\x01\x3D",
		"VOL0":    "\xA6\x01\x00\x00\x00\x04\x01\x44\x00",
		"VOL10":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x0A",
		"VOL20":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x14",
		"VOL30":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x1e",
		"VOL40":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x28",
		"VOL50":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x32",
		"VOL60":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x3C",
		"VOL70":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x46",
		"VOL80":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x50",
		"VOL90":   "\xA6\x01\x00\x00\x00\x04\x01\x44\x5A",
		"VOL100":  "\xA6\x01\x00\x00\x00\x04\x01\x44\x64",
		"PICNORM": "\xA6\x01\x00\x00\x00\x04\x01\x3A\x00",
		"PICCUST": "\xA6\x01\x00\x00\x00\x04\x01\x3A\x01",
		"PICREAL": "\xA6\x01\x00\x00\x00\x04\x01\x3A\x02",
		"PICFULL": "\xA6\x01\x00\x00\x00\x04\x01\x3A\x03",
		"PIC219":  "\xA6\x01\x00\x00\x00\x04\x01\x3A\x04",
		"PICDYN":  "\xA6\x01\x00\x00\x00\x04\x01\x3A\x05",
		"M-GAME":  "\xA6\x01\x00\x00\x00\x09\x01\x32\x64\x64\x64\x5A\x64\x00\x03",
		"M-OFF":   "\xA6\x01\x00\x00\x00\x09\x01\x32\x5F\x32\x32\x32\x32\x00\x03",
		"T-USER":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x00",
		"T-NATU":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x01",
		"T-3000":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x0D",
		"T-6500":  "\xA6\x01\x00\x00\x00\x04\x01\x34\x06",
		"T-10000": "\xA6\x01\x00\x00\x00\x04\x01\x34\x09",
	}

	val, ok := commands[*cmdFlag]
	file, err := os.OpenFile(*portFlag, os.O_RDWR, 0666)
	if (*helpFlag || !ok) {
		fmt.Fprintf(os.Stderr, "Cmd: %s\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Commands: \n")
		for k, _ := range commands {
			fmt.Fprintf(os.Stderr, "  %s\n", k)
		}
		os.Exit(0)
	}

	if err != nil {
		log.Fatal(err)
	}

	 _, _, errnop := syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(TCFLSH), uintptr(syscall.TCIOFLUSH))
	if (errnop != 0) {
		log.Fatal(errnop)	
	}

	var termios Termios
	 _, _, errnop = syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)))
	if (errnop != 0) {
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
	if (errnop != 0) {
		log.Fatal(errnop)	
	}

	cs := csum(val)
	fmt.Printf("CSUM: %x\n", cs)

	_, err = file.Write(append([]byte(val), cs))
	if err != nil {
		log.Fatal(err)
	}

	resp := make([]byte, 32)
	n, err := file.Read(resp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("OUT: ")
	printhex(resp, n)
}
