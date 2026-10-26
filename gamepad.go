package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	minAbsVal = -32767
	maxAbsVal = 32767
)

// VirtualDevice abstracts a platform-specific virtual gamepad (Linux uinput or Windows ViGEmBus).
type VirtualDevice interface {
	EmitButton(btn string, pressed bool) error
	EmitAxis(axes map[string]float64) error
	Close() error
}

// PlayerController holds the state and virtual device for a connected player.
type PlayerController struct {
	Username     string
	ControllerID int
	Device       VirtualDevice
	Axes         map[string]float64
}

// GamepadManager manages all active virtual controllers in memory.
type GamepadManager struct {
	mu          sync.Mutex
	players     map[string]*PlayerController
	lastAxisLog map[string]time.Time
	counter     int
	logFn       func(string)
}

func NewGamepadManager(logFn func(string)) *GamepadManager {
	if logFn == nil {
		logFn = func(string) {}
	}
	gm := &GamepadManager{
		players:     make(map[string]*PlayerController),
		lastAxisLog: make(map[string]time.Time),
		logFn:       logFn,
	}
	gm.logFn(platformInitMessage())
	gm.logFn("Waiting for players to connect...")
	return gm
}

func (gm *GamepadManager) RegisterPlayer(username string) (int, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if existing, ok := gm.players[username]; ok {
		return existing.ControllerID, nil
	}

	nextID := gm.counter + 1
	dev, err := newPlatformDevice(nextID)
	if err != nil {
		gm.logFn(fmt.Sprintf("[ERROR] Could not create virtual gamepad for '%s': %v", username, err))
		return 0, err
	}

	gm.counter = nextID
	pc := &PlayerController{
		Username:     username,
		ControllerID: nextID,
		Device:       dev,
		Axes: map[string]float64{
			"lx": 0.0,
			"ly": 0.0,
			"rx": 0.0,
			"ry": 0.0,
			"lt": 0.0,
			"rt": 0.0,
		},
	}
	gm.players[username] = pc
	gm.logFn(fmt.Sprintf("New player registered: '%s' as 'airInput-Controller%d'", username, nextID))
	return nextID, nil
}

func (gm *GamepadManager) DisconnectPlayer(username string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	pc, ok := gm.players[username]
	if !ok {
		return
	}
	if pc.Device != nil {
		_ = pc.Device.Close()
	}
	delete(gm.players, username)
	gm.logFn(fmt.Sprintf("Control for player '%s' (Controller%d) removed.", username, pc.ControllerID))
}

func (gm *GamepadManager) HandleButton(username, btnName string, state int) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	pc, ok := gm.players[username]
	if !ok || pc.Device == nil {
		return
	}

	pressed := state != 0
	if err := pc.Device.EmitButton(btnName, pressed); err == nil {
		status := "Released"
		if pressed {
			status = "Pressed"
		}
		gm.logFn(fmt.Sprintf("[%s (C%d)] Button: %s -> %s", username, pc.ControllerID, btnName, status))
	}
}

func (gm *GamepadManager) HandleAxis(username, axis string, val float64) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	pc, ok := gm.players[username]
	if !ok || pc.Device == nil {
		return
	}

	if _, validAxis := pc.Axes[axis]; !validAxis {
		return
	}

	if math.IsNaN(val) || math.IsInf(val, 0) {
		return
	}
	clamped := math.Max(-1.0, math.Min(1.0, val))
	pc.Axes[axis] = clamped

	if err := pc.Device.EmitAxis(pc.Axes); err == nil {
		now := time.Now()
		logKey := username + ":" + axis
		if clamped == 0.0 || now.Sub(gm.lastAxisLog[logKey]) >= 250*time.Millisecond {
			gm.lastAxisLog[logKey] = now
			gm.logFn(fmt.Sprintf("[%s (C%d)] Axis: %s -> %.2f", username, pc.ControllerID, axis, clamped))
		}
	}
}

func (gm *GamepadManager) SwapPlayers(userA, userB string) bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	pcA, okA := gm.players[userA]
	pcB, okB := gm.players[userB]
	if !okA || !okB {
		return false
	}

	pcA.Device, pcB.Device = pcB.Device, pcA.Device
	pcA.Axes, pcB.Axes = pcB.Axes, pcA.Axes
	pcA.ControllerID, pcB.ControllerID = pcB.ControllerID, pcA.ControllerID

	gm.logFn(fmt.Sprintf("[SYSTEM] Swapped controller devices for '%s' and '%s'", userA, userB))
	return true
}

func (gm *GamepadManager) CloseAll() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for username, pc := range gm.players {
		if pc.Device != nil {
			_ = pc.Device.Close()
		}
		delete(gm.players, username)
	}
}
