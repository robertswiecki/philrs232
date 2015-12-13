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
import "strconv"
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

func csum(v []byte) byte {
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
	custFlag := flag.String("cust", "", "Custom command, encoded as a C string")
	altFlag := flag.Bool("alt", false, "Don't prepend the \"\\xA6\\x01\\x00\\x00\\x00\" prefix")
	helpFlag := flag.Bool("help", false, "Help")
	speedFlag := flag.Int("speed", 9600, "ttyS speed")
	flag.Parse()

	commands := map[string]string{
		"ON":        "\x18\x02",
		"OFF":       "\x18\x01",
		"PIC-NORM":  "\x3A\x00",
		"PIC-CUST":  "\x3A\x01",
		"PIC-REAL":  "\x3A\x02",
		"PIC-FULL":  "\x3A\x03",
		"PIC-21-9":  "\x3A\x04",
		"PIC-DYN":   "\x3A\x05",
		"PIP-OFF":   "\x3C\x00\x00\x00\x00",
		"PIP-BL":    "\x3C\x01\x00\x00\x00",
		"PIP-TL":    "\x3C\x01\x01\x00\x00",
		"PIP-TR":    "\x3C\x01\x02\x00\x00",
		"PIP-BR":    "\x3C\x01\x03\x00\x00",
		"VOL0":      "\x44\x00",
		"VOL10":     "\x44\x0A",
		"VOL20":     "\x44\x14",
		"VOL30":     "\x44\x1e",
		"VOL40":     "\x44\x28",
		"VOL50":     "\x44\x32",
		"VOL60":     "\x44\x3C",
		"VOL70":     "\x44\x46",
		"VOL80":     "\x44\x50",
		"VOL90":     "\x44\x5A",
		"VOL100":    "\x44\x64",
/* It redefines current mode setting
		"M-NORM":    "\x32\x32\x32\x32\x32\x32",
		"M-MOVIE":   "\x32\x64\x32\x32\x5A\x32",
*/
		"REP-INPUT": "\xAD",
		"IN-VGA":    "\xAC\x05\x00\x01\x00",
		"IN-HDMI":   "\xAC\x06\x02\x01\x00",
		"IN-MHDMI":  "\xAC\x06\x03\x01\x00",
		"IN-DP":     "\xAC\x09\x04\x01\x00",
		"IN-MDP":    "\xAC\x09\x05\x01\x00",
	}

	val, ok := commands[*cmdFlag]
	if *custFlag != "" {
		convval := fmt.Sprintf("\"%s\"", *custFlag)
		convval, err := strconv.Unquote(convval)
		if err != nil {
			log.Fatal("strconv.Unquote(\"", *custFlag, "\"): ", err)
		}
		ok = true
		val = convval
	}

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

	termios.Iflag |= syscall.IGNPAR

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

	request := []byte("")
	if !*altFlag {
		request = append(request, []byte("\xA6\x01\x00\x00\x00")...)
	}
	if *altFlag {
		request = append(request, byte(len(val)+4))
	} else {
		request = append(request, byte(len(val)+2))
	}
	request = append(request, '\x01')
	if *altFlag {
		request = append(request, byte(0))
	}
	request = append(request, []byte(val)...)
	request = append(request, csum(request))

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
