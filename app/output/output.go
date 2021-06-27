package output

import (
	"bytes"
	"fmt"
	"io"
	"mrz.io/hkswitch/app/config"
)

func Prefix(s string, n int, sep string) string {
	return fmt.Sprintf("%-"+fmt.Sprintf("%d", n)+"s%s", s, sep)
}

func WithPrefix(w io.Writer, prefix string) io.Writer {
	return &prefixer{w: w, prefix: prefix, writePrefix: true}
}

type prefixer struct {
	w           io.Writer
	prefix      string
	writePrefix bool
}

func (p *prefixer) Write(data []byte) (n int, err error) {
	buf := bytes.NewBuffer(nil)

	for _, b := range data {
		if p.writePrefix {
			buf.WriteString(p.prefix)
			p.writePrefix = false
		}

		buf.WriteByte(b)

		if b == '\n' {
			p.writePrefix = true
		}
	}

	dataLen := len(data)

	if n, err := p.w.Write(buf.Bytes()); err != nil {
		if n <= dataLen {
			return n, err
		} else {
			return dataLen, err
		}
	}

	return dataLen, nil
}

type Prefixer struct {
	stdout io.Writer
	stderr io.Writer

	minLen int
	sep    string
}

func NewPrefixer(stdout io.Writer, stderr io.Writer, minLen int, sep string) *Prefixer {
	return &Prefixer{stdout: stdout, stderr: stderr, minLen: minLen, sep: sep}
}

func (p Prefixer) Stdout(svc config.Service) io.Writer {
	return WithPrefix(p.stdout, Prefix(svc.Name, p.minLen, p.sep))
}

func (p Prefixer) Stderr(svc config.Service) io.Writer {
	return WithPrefix(p.stderr, Prefix(svc.Name, p.minLen, p.sep))
}

func FindPrefixSize(minLen int, names ...string) int {
	for _, n := range names {
		if len(n) > minLen {
			minLen = len(n)
		}
	}

	return minLen
}
