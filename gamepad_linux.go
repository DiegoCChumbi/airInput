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

	uiSetEvBit  = 0x40045564
	uiSetKeyBit = 0x40045565
	uiSetAbsBit = 0x40045567
	uiDevCreate = 0x5501
	uiDevDestroy = 0x5502

	busUSB = 0x03

	absX  = 0x00
	absY  = 0x01
	absRX = 0x03
	absRY = 0x04
)

// Note: In the Linux kernel (input-event-codes.h), 0x133 is BTN_NORTH (aliased as BTN_X)
// and 0x134 is BTN_WEST (aliased as BTN_Y). Modern browsers (W3C Gamepad API) and SDL2 map
// BTN_WEST (0x134) to X (Left, buttons[2]) and BTN_NORTH (0x133) to Y (Top, buttons[3]).
var linuxBtnMap = map[string]uint16{
	"A":      0x130, // BTN_SOUTH / BTN_A
	"B":      0x131, // BTN_EAST  / BTN_B
	"X":      0x134, // BTN_WEST  (mapped to X)
	"Y":      0x133, // BTN_NORTH (mapped to Y)
	"L":      0x136, // BTN_TL
	"R":      0x137, // BTN_TR
	"L2":     0x138, // BTN_TL2
	"R2":     0x139, // BTN_TR2
	"SELECT": 0x13a, // BTN_SELECT
	"START":  0x13b, // BTN_START
	"L3":     0x13d, // BTN_THUMBL
	"R3":     0x13e, // BTN_THUMBR
	"UP":     0x220, // BTN_DPAD_UP
	"DOWN":   0x221, // BTN_DPAD_DOWN
	"LEFT":   0x222, // BTN_DPAD_LEFT
	"RIGHT":  0x223, // BTN_DPAD_RIGHT
}

var linuxAllKeyCodes = []uint16{
	0x130, 0x131, 0x133, 0x134,
	0x136, 0x137, 0x138, 0x139,
	0x13a, 0x13b, 0x13d, 0x13e,
	0x220, 0x221, 0x222, 0x223,
}

var linuxAllAbsCodes = []uint16{
	absX, absY, absRX, absRY,
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
}

func platformInitMessage() string {
	return "Virtual Gamepad Server started (Linux native uinput in Go)."
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
	name := fmt.Sprintf("airInput-Controller%d", controllerNum)
	copy(dev.Name[:79], []byte(name))
	dev.IDBusType = busUSB
	dev.IDVendor = 0x0000
	dev.IDProduct = 0x0000
	dev.IDVersion = 0x0001

	for _, absCode := range linuxAllAbsCodes {
		if err := ioctl(fd, uiSetAbsBit, uintptr(absCode)); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("UI_SET_ABSBIT failed: %w", err)
		}
		dev.Absmin[absCode] = minAbsVal
		dev.Absmax[absCode] = maxAbsVal
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

	// Center all 4 axes immediately so OS/browsers never read uninitialized values
	for _, absCode := range linuxAllAbsCodes {
		_ = d.writeEvent(evAbs, absCode, 0, false)
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
