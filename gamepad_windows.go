//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	vigemErrorNone = 0x20000000
)

var winBtnMap = map[string]uint16{
	"UP":     0x0001, // XUSB_GAMEPAD_DPAD_UP
	"DOWN":   0x0002, // XUSB_GAMEPAD_DPAD_DOWN
	"LEFT":   0x0004, // XUSB_GAMEPAD_DPAD_LEFT
	"RIGHT":  0x0008, // XUSB_GAMEPAD_DPAD_RIGHT
	"START":  0x0010, // XUSB_GAMEPAD_START
	"SELECT": 0x0020, // XUSB_GAMEPAD_BACK
	"L3":     0x0040, // XUSB_GAMEPAD_LEFT_THUMB
	"R3":     0x0080, // XUSB_GAMEPAD_RIGHT_THUMB
	"L":      0x0100, // XUSB_GAMEPAD_LEFT_SHOULDER
	"R":      0x0200, // XUSB_GAMEPAD_RIGHT_SHOULDER
	"A":      0x1000, // XUSB_GAMEPAD_A
	"B":      0x2000, // XUSB_GAMEPAD_B
	"X":      0x4000, // XUSB_GAMEPAD_X
	"Y":      0x8000, // XUSB_GAMEPAD_Y
}

// xusbReport matches XUSB_REPORT (12 bytes) from ViGEm/Common.h.
type xusbReport struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

var (
	vigemDLL             = syscall.NewLazyDLL("ViGEmClient.dll")
	procAlloc            = vigemDLL.NewProc("vigem_alloc")
	procFree             = vigemDLL.NewProc("vigem_free")
	procConnect          = vigemDLL.NewProc("vigem_connect")
	procDisconnect       = vigemDLL.NewProc("vigem_disconnect")
	procTargetX360Alloc  = vigemDLL.NewProc("vigem_target_x360_alloc")
	procTargetFree       = vigemDLL.NewProc("vigem_target_free")
	procTargetAdd        = vigemDLL.NewProc("vigem_target_add")
	procTargetRemove     = vigemDLL.NewProc("vigem_target_remove")
	procTargetX360Update = vigemDLL.NewProc("vigem_target_x360_update")
)

type winViGEmDevice struct {
	client uintptr
	target uintptr
	report xusbReport
}

func platformInitMessage() string {
	return "Virtual Gamepad Server started (Windows ViGEmBus in Go - EXPERIMENTAL / UNTESTED)."
}

func newPlatformDevice(controllerNum int) (VirtualDevice, error) {
	if err := vigemDLL.Load(); err != nil {
		return nil, fmt.Errorf("ViGEmClient.dll no encontrado (%v) — instala ViGEmBus y coloca ViGEmClient.dll junto a airInput.exe", err)
	}

	client, _, _ := procAlloc.Call()
	if client == 0 {
		return nil, fmt.Errorf("vigem_alloc falló")
	}

	ret, _, _ := procConnect.Call(client)
	if uint32(ret) != vigemErrorNone {
		procFree.Call(client)
		return nil, fmt.Errorf("vigem_connect falló (0x%X) — verifica que el driver ViGEmBus esté instalado", uint32(ret))
	}

	target, _, _ := procTargetX360Alloc.Call()
	if target == 0 {
		procDisconnect.Call(client)
		procFree.Call(client)
		return nil, fmt.Errorf("vigem_target_x360_alloc falló")
	}

	ret, _, _ = procTargetAdd.Call(client, target)
	if uint32(ret) != vigemErrorNone {
		procTargetFree.Call(target)
		procDisconnect.Call(client)
		procFree.Call(client)
		return nil, fmt.Errorf("vigem_target_add falló (0x%X)", uint32(ret))
	}

	d := &winViGEmDevice{
		client: client,
		target: target,
	}
	_ = d.sendReport()
	return d, nil
}

// sendReport sends the 12-byte XUSB_REPORT to ViGEmClient.
// On 64-bit Windows (Microsoft x64 ABI), structs whose size is not 1, 2, 4, or 8 bytes
// are passed by reference (pointer to a caller-allocated copy).
func (d *winViGEmDevice) sendReport() error {
	if d.client == 0 || d.target == 0 {
		return fmt.Errorf("device closed")
	}
	reportCopy := d.report
	ret, _, _ := procTargetX360Update.Call(d.client, d.target, uintptr(unsafe.Pointer(&reportCopy)))
	if uint32(ret) != vigemErrorNone {
		return fmt.Errorf("vigem_target_x360_update failed: 0x%X", uint32(ret))
	}
	return nil
}

func (d *winViGEmDevice) EmitButton(btn string, pressed bool) error {
	switch btn {
	case "L2":
		if pressed {
			d.report.LeftTrigger = 255
		} else {
			d.report.LeftTrigger = 0
		}
	case "R2":
		if pressed {
			d.report.RightTrigger = 255
		} else {
			d.report.RightTrigger = 0
		}
	default:
		mask, ok := winBtnMap[btn]
		if !ok {
			return fmt.Errorf("unknown button: %s", btn)
		}
		if pressed {
			d.report.Buttons |= mask
		} else {
			d.report.Buttons &^= mask
		}
	}
	return d.sendReport()
}

func clampInt16(v float64) int16 {
	scaled := int32(v * float64(maxAbsVal))
	if scaled < minAbsVal {
		scaled = minAbsVal
	} else if scaled > maxAbsVal {
		scaled = maxAbsVal
	}
	return int16(scaled)
}

func (d *winViGEmDevice) EmitAxis(axes map[string]float64) error {
	// Client sends -1.0 for UP and +1.0 for DOWN (standard screen/evdev convention).
	// XInput (XUSB_REPORT) expects +32767 for UP and -32767 for DOWN, so negate ly and ry.
	d.report.ThumbLX = clampInt16(axes["lx"])
	d.report.ThumbLY = clampInt16(-axes["ly"])
	d.report.ThumbRX = clampInt16(axes["rx"])
	d.report.ThumbRY = clampInt16(-axes["ry"])
	return d.sendReport()
}

func (d *winViGEmDevice) Close() error {
	if d.client == 0 {
		return nil
	}
	if d.target != 0 {
		procTargetRemove.Call(d.client, d.target)
		procTargetFree.Call(d.target)
		d.target = 0
	}
	procDisconnect.Call(d.client)
	procFree.Call(d.client)
	d.client = 0
	return nil
}
