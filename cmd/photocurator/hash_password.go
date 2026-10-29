package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"photocurator/internal/auth"
)

// hashPassword reads one password and prints only its hash, so stdout can be piped into users.conf.
func hashPassword(in *os.File, out, errOut io.Writer) int {
	pw, err := readPassword(in, errOut)
	if err == nil && pw == "" {
		err = errors.New("empty password")
	}
	var hash string
	if err == nil {
		hash, err = auth.HashPassword(pw)
	}
	if err != nil {
		fmt.Fprintln(errOut, "hash-password:", err)
		return 1
	}
	fmt.Fprintln(out, hash)
	return 0
}

func readPassword(in *os.File, errOut io.Writer) (string, error) {
	if term.IsTerminal(int(in.Fd())) {
		fmt.Fprint(errOut, "Password: ")
		b, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(errOut)
		return string(b), err
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
