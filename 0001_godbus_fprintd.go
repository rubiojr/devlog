// Using fprintd DBUS interface to use the built-in fingerprint scanner available in some laptops.
//
// Pre-requisites:
//
// Linux only.
// The fingerprint scanner needs to be properly configured and working.
//
// Related material:
//
// - https://fprint.freedesktop.org/fprintd-dev/Device.html#Device::VerifyStatus
// - https://help.gnome.org/users/gnome-help/stable/session-fingerprint.html.en
// - https://github.com/godbus/dbus
//
// Useful software for debugging:
// - bustle: https://github.com/freedesktop/bustle
// - d-feet: https://wiki.gnome.org/Apps/DFeet
//
package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

const VERIFY_ATTEMPTS = 3

func main() {
	conn, err := dbus.SystemBus()
	if err != nil {
		errexit(err)
	}
	defer conn.Close()

	err = conn.AddMatchSignal(dbus.WithMatchObjectPath("/net/reactivated/Fprint/Device/0"))
	if err != nil {
		errexit(err)
	}

	obj := conn.Object("net.reactivated.Fprint", "/net/reactivated/Fprint/Device/0")

	verify(conn, obj, 0)
}

func verify(conn *dbus.Conn, obj dbus.BusObject, attempts int) {
	call := obj.Call("net.reactivated.Fprint.Device.Claim", 0, "rubiojr")
	if call.Err != nil {
		errexit(call.Err)
	}

	release := func() {
		call = obj.Call("net.reactivated.Fprint.Device.Release", 0)
		if call.Err != nil {
			errexit(call.Err)
		}
	}
	defer release()

	call = obj.Call("net.reactivated.Fprint.Device.VerifyStart", 0, "any")
	if call.Err != nil {
		errexit(call.Err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for v := range c {
		if attempts == VERIFY_ATTEMPTS {
			fmt.Println("Max attempts exhausted.")
			failure()
		}

		if v.Name == "net.reactivated.Fprint.Device.VerifyStatus" {
			switch v.Body[0] {
			case "verify-match":
				success()
			case "verify-no-match":
				fmt.Println("Verification failed. Retrying...")
				release()
				verify(conn, obj, attempts+1)
			}
			attempts += 1
			fmt.Println("Please retry scanning your finger...")
		}
	}
}

func success() {
	fmt.Println("Verification successful ✅")
	os.Exit(0)
}

func failure() {
	fmt.Println("Verification failed ❌")
	os.Exit(1)
}

func errexit(err error) {
	fmt.Println("An error ocurred: %v", err)
	os.Exit(1)
}
