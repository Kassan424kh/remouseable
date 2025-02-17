// This file is part of remouseable.
//
// remouseable is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License version 3 as published
// by the Free Software Foundation.
//
// remouseable is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with remouseable.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"syscall"

	remouseable "github.com/kevinconway/remouseable/pkg"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/widget"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

var closeApp bool = false

func connectToRemarkableTablet(closeApp bool) {
	driver := &remouseable.RobotgoDriver{}

	fs := flag.NewFlagSet("remouseable", flag.ExitOnError)
	orientation := fs.String("orientation", "right", "Orientation of the tablet. Choices are vertical, right, and left")
	tabletHeight := fs.Int("tablet-height", remouseable.DefaultTabletHeight, "The max units per millimeter for the hight of the tablet. Probably don't change this.")
	tabletWidth := fs.Int("tablet-width", remouseable.DefaultTabletWidth, "The max units per millimeter for the width of the tablet. Probably don't change this.")
	tmpScreenWidth, tmpScreenHeight, _ := driver.GetSize()
	screenHeight := fs.Int("screen-height", tmpScreenHeight, "The max units per millimeter of the host screen height. Probably don't change this.")
	screenWidth := fs.Int("screen-width", tmpScreenWidth, "The max units per millimeter of the host screen width. Probably don't change this.")
	sshIP := fs.String("ssh-ip", "192.168.0.102:22", "The host and port of a tablet.")
	sshUser := fs.String("ssh-user", "root", "The ssh username to use when logging into the tablet.")
	sshPassword := fs.String("ssh-password", "iuhliwJF8P", "An optional password to use when ssh-ing into the tablet. Use - for a prompt rather than entering a value. If not given then public/private keypair authentication is used.")
	sshSocket := fs.String("ssh-socket", os.Getenv("SSH_AUTH_SOCK"), "Path to the SSH auth socket. This must not be empty if using public/private keypair authentication.")
	evtFile := fs.String("event-file", "/dev/input/event1", "The path on the tablet from which to read evdev events. Probably don't change this.")
	debugEvents := fs.Bool("debug-events", false, "Stream hardware events from the tablet instead of acting as a mouse. This is for debugging.")
	disableDrag := fs.Bool("disable-drag-event", false, "Disable use of the custom OSX drag event. Only use this drawing on an Apple device is not working as expected.")
	pressureThreshold := fs.Int("pressure-threshold", 1000, "Change the click detection sensitivity. 1000 is when the pen makes contact with the tablet. Set higher to require more pen pressure for a click.")
	_ = fs.Parse(os.Args[1:])

	if *sshPassword == "-" {
		fmt.Print("Enter Password: ")
		pwd, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println(err)
		}
		*sshPassword = string(pwd)
	}
	sshConfig := &ssh.ClientConfig{
		User: *sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(*sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if *sshPassword == "" {
		agentFd, err := net.Dial("unix", *sshSocket)
		if err != nil {
			fmt.Println(err)
		}
		defer agentFd.Close()

		agentSigner := agent.NewClient(agentFd)

		sshConfig = &ssh.ClientConfig{
			User: *sshUser,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeysCallback(agentSigner.Signers),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	}

	client, err := ssh.Dial("tcp", *sshIP, sshConfig)
	if err != nil {
		fmt.Println(err)
	}

	sesh, err := client.NewSession()
	if err != nil {
		fmt.Println(err)
	}
	defer sesh.Close()

	pipe, err := sesh.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	if err = sesh.Start(fmt.Sprintf("cat %s", *evtFile)); err != nil {
		fmt.Println(err)
	}
	if *debugEvents {
		it := &remouseable.SelectingEvdevIterator{
			Wrapped: &remouseable.FileEvdevIterator{
				Source: ioutil.NopCloser(pipe),
			},
			Selection: []uint16{remouseable.EV_ABS},
		}
		defer it.Close()
		fmt.Println("remouseable connected and running.")
		for it.Next() {
			evt := it.Current()
			evtype := remouseable.EVMap[evt.Type]
			evcode := remouseable.CodeString(evt.Type, evt.Code)
			fmt.Printf(
				`{"eventType": %d, "eventTypeName": "%s", "eventCode": %d, "eventCodeName": "%s", "eventValue": %d}`,
				evt.Type, evtype, evt.Code, evcode, evt.Value,
			)
			fmt.Print("\n")
		}
		if err = it.Close(); err != nil {
			fmt.Println(err.Error())
		}
		return
	}

	it := &remouseable.SelectingEvdevIterator{
		Wrapped: &remouseable.FileEvdevIterator{
			Source: ioutil.NopCloser(pipe),
		},
		Selection: []uint16{remouseable.EV_ABS},
	}
	defer it.Close()

	var sm remouseable.StateMachine = &remouseable.DraggingEvdevStateMachine{
		EvdevStateMachine: &remouseable.EvdevStateMachine{
			Iterator:          it,
			PressureThreshold: *pressureThreshold,
		},
	}
	if *disableDrag {
		sm = &remouseable.EvdevStateMachine{
			Iterator:          it,
			PressureThreshold: *pressureThreshold,
		}
	}
	defer sm.Close()

	var sc remouseable.PositionScaler
	switch *orientation {
	case "right":
		sc = &remouseable.RightPositionScaler{
			TabletWidth:  *tabletWidth,
			TabletHeight: *tabletHeight,
			ScreenWidth:  *screenWidth,
			ScreenHeight: *screenHeight,
		}
	case "left":
		sc = &remouseable.LeftPositionScaler{
			TabletWidth:  *tabletWidth,
			TabletHeight: *tabletHeight,
			ScreenWidth:  *screenWidth,
			ScreenHeight: *screenHeight,
		}
	case "vertical":
		sc = &remouseable.VerticalPositionScaler{
			TabletWidth:  *tabletWidth,
			TabletHeight: *tabletHeight,
			ScreenWidth:  *screenWidth,
			ScreenHeight: *screenHeight,
		}
	default:
		fmt.Println(fmt.Sprintf("unknown orienation selection %s", *orientation))
	}

	rt := &remouseable.Runtime{
		PositionScaler: sc,
		StateMachine:   sm,
		Driver:         driver,
	}

	fmt.Println("remouseable connected and running.")
	for rt.Next() {
	}
	if err = rt.Close(); err != nil {
		fmt.Println("***********")
		fmt.Println(err)
	}

}

func main() {
	a := app.New()

	imageResourse, err := fyne.LoadResourceFromPath("remouseable_logo_release.png")
	if err != nil {
		fmt.Println(err)
	}
	a.SetIcon(imageResourse)
	a.Settings().SetTheme(theme.DarkTheme())

	win := a.NewWindow("Hello World")
	go connectToRemarkableTablet(closeApp)

	r := widget.NewLabel("Hello World")
	win.SetContent(widget.NewVBox(
		r,
		widget.NewButton("Connect", func() {
			closeApp = false
		}),
		widget.NewButton("Reconnect", func() {
			r.TextStyle.Italic = true
			r.SetText("text")
			closeApp = true

		}),
		widget.NewButton("Quit", func() {
			closeApp = true
			a.Quit()
		}),
	))

	win.SetFixedSize(true)
	win.Resize(fyne.NewSize(200, 400))

	var resourceImagePng = canvas.NewImageFromFile("remouseable_logo.png")
	win.SetIcon(resourceImagePng.Resource)

	win.ShowAndRun()

}
