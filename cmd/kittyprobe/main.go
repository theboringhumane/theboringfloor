// Command kittyprobe asks the REAL terminal which kitty graphics frame
// shapes it accepts, by emitting q=1 (respond) probe transmissions and
// printing the terminal's own responses. It exists to debug "the zenbu
// lane's frames don't paint" without guessing: run it inside the target
// terminal (kitty/ghostty) and read the verdicts.
//
// Usage: kittyprobe
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func probe(fd int, tty *os.File, name string, keys string, payload []byte) {
	b64 := base64.StdEncoding.EncodeToString(payload)
	cmd := fmt.Sprintf("\x1b_G%s;%s\x1b\\", keys, b64)
	// drain anything pending
	buf := make([]byte, 64*1024)
	for {
		tv := time.Now().Add(60 * time.Millisecond)
		_ = tty.SetReadDeadline(tv)
		if _, err := tty.Read(buf); err != nil {
			break
		}
	}
	if _, err := tty.WriteString(cmd); err != nil {
		fmt.Printf("%-34s -> write error: %v\n", name, err)
		return
	}
	deadline := time.Now().Add(600 * time.Millisecond)
	var resp bytes.Buffer
	for time.Now().Before(deadline) {
		_ = tty.SetReadDeadline(time.Now().Add(120 * time.Millisecond))
		n, err := tty.Read(buf)
		if n > 0 {
			resp.Write(buf[:n])
		}
		if err != nil && resp.Len() > 0 {
			break
		}
	}
	out := bytes.ReplaceAll(resp.Bytes(), []byte{0x1b}, []byte("<ESC>"))
	if len(out) == 0 {
		out = []byte("(no response)")
	}
	fmt.Printf("%-34s -> %s\n", name, out)
}

func zlibRGBA() []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, _ = w.Write([]byte{255, 0, 0, 255}) // 1x1 red RGBA
	_ = w.Close()
	return b.Bytes()
}

func pngRGBA() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kittyprobe: /dev/tty:", err)
		os.Exit(1)
	}
	defer tty.Close()
	fd := int(tty.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kittyprobe: get termios:", err)
		os.Exit(1)
	}
	raw := *oldState
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &raw); err != nil {
		fmt.Fprintln(os.Stderr, "kittyprobe: raw mode:", err)
		os.Exit(1)
	}
	defer unix.IoctlSetTermios(fd, unix.TIOCSETA, oldState)

	z := zlibRGBA()
	p := pngRGBA()
	type t struct{ name, keys string; payload []byte }
	tests := []t{
		{"A f=32 o=z (child verbatim)", "a=T,t=d,f=32,o=z,s=1,v=1,q=1,i=9101", z},
		{"B f=32 o=z + c/r", "a=T,t=d,f=32,o=z,s=1,v=1,q=1,c=10,r=5,i=9102", z},
		{"C f=32 no zlib", "a=T,t=d,f=32,s=1,v=1,q=1,i=9103", []byte{255, 0, 0, 255}},
		{"D f=100 png", "a=T,t=d,f=100,q=1,i=9104", p},
		{"E f=100 png + c/r", "a=T,t=d,f=100,q=1,c=10,r=5,i=9105", p},
		{"F office-exact (C=1 p=1 c/r)", "a=T,t=d,q=1,C=1,f=32,o=z,s=1,v=1,c=10,r=5,p=1,i=9106", z},
	}
	for _, tc := range tests {
		probe(fd, tty, tc.name, tc.keys, tc.payload)
	}
	for i := 9101; i <= 9106; i++ {
		_, _ = tty.WriteString(fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\", i))
	}
	fmt.Println("done — every line above is the terminal's own answer (OK = accepted; anything else = the rejection reason)")
}
