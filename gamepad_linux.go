//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
)

const (
	evSyn = 0x00
	evKey = 0x01
	evAbs = 0x03

	synReport = 0x00

	uiSetEvBit   = 0x40045564
	uiSetKeyBit  = 0x40045565
	uiSetAbsBit  = 0x40045567
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502

	busUSB = 0x03

	absX     = 0x00
	absY     = 0x01
	absZ     = 0x02
	absRX    = 0x03
	absRY    = 0x04
	absRZ    = 0x05
	absHat0X = 0x10
	absHat0Y = 0x11
)

// Standard Linux xpad (Microsoft X-Box 360 pad, 045e:028e v0110) button mapping.
// In xpad/SDL2/Chromium standard mappings for 045e:028e:
// b0=0x130(A), b1=0x131(B), b2=0x133(X), b3=0x134(Y), b4=0x136(L), b5=0x137(R),
// b6=0x13a(SELECT), b7=0x13b(START), b8=0x13c(MODE), b9=0x13d(L3), b10=0x13e(R3).
// L2/R2 are emitted on ABS_Z (a2) and ABS_RZ (a5), and D-Pad on ABS_HAT0X/Y.
var linuxBtnMap = map[string]uint16{
	"A":      0x130, // BTN_A (b0)
	"B":      0x131, // BTN_B (b1)
	"X":      0x133, // BTN_X (b2)
	"Y":      0x134, // BTN_Y (b3)
	"L":      0x136, // BTN_TL (b4)
	"R":      0x137, // BTN_TR (b5)
	"SELECT": 0x13a, // BTN_SELECT (b6)
	"START":  0x13b, // BTN_START (b7)
	"L3":     0x13d, // BTN_THUMBL (b9)
	"R3":     0x13e, // BTN_THUMBR (b10)
}

var linuxAllKeyCodes = []uint16{
	0x130, // BTN_A
	0x131, // BTN_B
	0x133, // BTN_X
	0x134, // BTN_Y
	0x136, // BTN_TL
	0x137, // BTN_TR
	0x13a, // BTN_SELECT
	0x13b, // BTN_START
	0x13c, // BTN_MODE (Guide)
	0x13d, // BTN_THUMBL
	0x13e, // BTN_THUMBR
}

type absRange struct {
	code uint16
	min  int32
	max  int32
}

var linuxAllAbsRanges = []absRange{
	{absX, minAbsVal, maxAbsVal},
	{absY, minAbsVal, maxAbsVal},
	{absZ, 0, 255},
	{absRX, minAbsVal, maxAbsVal},
	{absRY, minAbsVal, maxAbsVal},
	{absRZ, 0, 255},
	{absHat0X, -1, 1},
	{absHat0Y, -1, 1},
}

// uinputUserDev matches the C struct uinput_user_dev in <linux/uinput.h> (1116 bytes).
type uinputUserDev struct {
	Name         [80]byte
	IDBusType    uint16
	IDVendor     uint16
	IDProduct    uint16
	IDVersion    uint16
	FFEffectsMax uint32
	Absmax       [64]int32
	Absmin       [64]int32
	Absfuzz      [64]int32
	Absflat      [64]int32
}

// inputEvent matches the C struct input_event in <linux/input.h>.
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type linuxUInputDevice struct {
	file     *os.File
	lastAxes map[uint16]int32
	dpadUp   bool
	dpadDown bool
	dpadLeft bool
	dpadRight bool
}

func platformInitMessage() string {
	return "Virtual Gamepad Server started (Linux native uinput Xbox 360 in Go)."
}

func ioctl(fd uintptr, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

func newPlatformDevice(controllerNum int) (VirtualDevice, error) {
	uinputPaths := []string{"/dev/uinput", "/dev/input/uinput"}
	var f *os.File
	var openErr error

	for _, p := range uinputPaths {
		if _, err := os.Stat(p); err == nil {
			f, openErr = os.OpenFile(p, os.O_WRONLY|syscall.O_NONBLOCK, 0660)
			if openErr == nil {
				break
			}
		}
	}
	if f == nil {
		if openErr != nil {
			return nil, fmt.Errorf("no se pudo abrir /dev/uinput (%v) — ejecuta con sudo", openErr)
		}
		return nil, fmt.Errorf("/dev/uinput no encontrado (ejecuta: sudo modprobe uinput)")
	}

	fd := f.Fd()

	for _, ev := range []uintptr{evSyn, evKey, evAbs} {
		if err := ioctl(fd, uiSetEvBit, ev); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("UI_SET_EVBIT failed: %w", err)
		}
	}

	for _, key := range linuxAllKeyCodes {
		if err := ioctl(fd, uiSetKeyBit, uintptr(key)); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("UI_SET_KEYBIT failed: %w", err)
		}
	}

	var dev uinputUserDev
	name := fmt.Sprintf("Microsoft X-Box 360 pad %d", controllerNum)
	copy(dev.Name[:79], []byte(name))
	dev.IDBusType = busUSB
	dev.IDVendor = 0x045e  // Microsoft Corp.
	dev.IDProduct = 0x028e // Xbox 360 Controller
	dev.IDVersion = 0x0110 // Standard xpad version

	for _, ar := range linuxAllAbsRanges {
		if err := ioctl(fd, uiSetAbsBit, uintptr(ar.code)); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("UI_SET_ABSBIT failed: %w", err)
		}
		dev.Absmin[ar.code] = ar.min
		dev.Absmax[ar.code] = ar.max
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &dev); err != nil {
		_ = f.Close()
		return nil, err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("error writing uinput_user_dev: %w", err)
	}

	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("UI_DEV_CREATE failed: %w", err)
	}

	d := &linuxUInputDevice{
		file:     f,
		lastAxes: make(map[uint16]int32),
	}

	// Center all sticks, triggers, and hat axes immediately on creation
	for _, ar := range linuxAllAbsRanges {
		_ = d.writeEvent(evAbs, ar.code, 0, false)
	}
	_ = d.writeEvent(evSyn, synReport, 0, false)

	return d, nil
}

func (d *linuxUInputDevice) writeEvent(evType, evCode uint16, value int32, syn bool) error {
	if d.file == nil {
		return fmt.Errorf("device closed")
	}
	var tv syscall.Timeval
	_ = syscall.Gettimeofday(&tv)

	var buf bytes.Buffer
	ev := inputEvent{
		Time:  tv,
		Type:  evType,
		Code:  evCode,
		Value: value,
	}
	_ = binary.Write(&buf, binary.LittleEndian, &ev)

	if syn {
		synEv := inputEvent{
			Time:  tv,
			Type:  evSyn,
			Code:  synReport,
			Value: 0,
		}
		_ = binary.Write(&buf, binary.LittleEndian, &synEv)
	}

	_, err := d.file.Write(buf.Bytes())
	return err
}

func (d *linuxUInputDevice) EmitButton(btn string, pressed bool) error {
	switch btn {
	case "L2":
		val := int32(0)
		if pressed {
			val = 255
		}
		return d.writeEvent(evAbs, absZ, val, true)

	case "R2":
		val := int32(0)
		if pressed {
			val = 255
		}
		return d.writeEvent(evAbs, absRZ, val, true)

	case "UP", "DOWN":
		if btn == "UP" {
			d.dpadUp = pressed
		} else {
			d.dpadDown = pressed
		}
		hatY := int32(0)
		if d.dpadUp && !d.dpadDown {
			hatY = -1
		} else if d.dpadDown && !d.dpadUp {
			hatY = 1
		}
		return d.writeEvent(evAbs, absHat0Y, hatY, true)

	case "LEFT", "RIGHT":
		if btn == "LEFT" {
			d.dpadLeft = pressed
		} else {
			d.dpadRight = pressed
		}
		hatX := int32(0)
		if d.dpadLeft && !d.dpadRight {
			hatX = -1
		} else if d.dpadRight && !d.dpadLeft {
			hatX = 1
		}
		return d.writeEvent(evAbs, absHat0X, hatX, true)
	}

	code, ok := linuxBtnMap[btn]
	if !ok {
		return fmt.Errorf("unknown button: %s", btn)
	}
	val := int32(0)
	if pressed {
		val = 1
	}
	return d.writeEvent(evKey, code, val, true)
}

func (d *linuxUInputDevice) EmitAxis(axes map[string]float64) error {
	targets := []struct {
		key  string
		code uint16
	}{
		{"lx", absX},
		{"ly", absY},
		{"rx", absRX},
		{"ry", absRY},
	}

	changed := false
	for _, t := range targets {
		scaled := int32(axes[t.key] * float64(maxAbsVal))
		if scaled < minAbsVal {
			scaled = minAbsVal
		} else if scaled > maxAbsVal {
			scaled = maxAbsVal
		}
		if prev, exists := d.lastAxes[t.code]; !exists || prev != scaled {
			d.lastAxes[t.code] = scaled
			if err := d.writeEvent(evAbs, t.code, scaled, false); err != nil {
				return err
			}
			changed = true
		}
	}

	if changed {
		return d.writeEvent(evSyn, synReport, 0, false)
	}
	return nil
}

func (d *linuxUInputDevice) Close() error {
	if d.file == nil {
		return nil
	}
	_ = ioctl(d.file.Fd(), uiDevDestroy, 0)
	err := d.file.Close()
	d.file = nil
	return err
}
