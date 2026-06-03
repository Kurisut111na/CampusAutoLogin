package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
)

// =============================================================================
// System Tray Icon
// =============================================================================

// TrayIcon manages the system tray notification area icon.
type TrayIcon struct {
	ni            *walk.NotifyIcon
	iconConnected *walk.Icon
	iconLoggedIn  *walk.Icon
	iconLost      *walk.Icon

	loggedIn  bool
	connected bool

	// Callbacks
	onShowWindow func()
	onReconnect  func()
	onOpenLogDir func()
	onQuit       func()
}

// NewTrayIcon creates a new system tray icon.
// The form parameter is the main window (required by walk.NotifyIcon).
func NewTrayIcon(form walk.Form) (*TrayIcon, error) {
	ti := &TrayIcon{}

	// Generate colored icons
	var err error
	ti.iconConnected, err = generateColorIcon(0x00, 0xCC, 0x00) // green
	if err != nil {
		return nil, fmt.Errorf("generate green icon: %w", err)
	}
	ti.iconLoggedIn, err = generateColorIcon(0x00, 0x88, 0xCC) // blue
	if err != nil {
		return nil, fmt.Errorf("generate blue icon: %w", err)
	}
	ti.iconLost, err = generateColorIcon(0xCC, 0x33, 0x33) // red
	if err != nil {
		return nil, fmt.Errorf("generate red icon: %w", err)
	}

	// Create the notify icon
	ni, err := walk.NewNotifyIcon(form)
	if err != nil {
		return nil, fmt.Errorf("NewNotifyIcon: %w", err)
	}
	ti.ni = ni

	ni.SetToolTip("Campus Auto Login")
	ni.SetIcon(ti.iconLoggedIn) // start with blue (logged in state default)

	// Build the context menu
	ti.buildMenu()

	// Double-click: show window
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if ti.onShowWindow != nil {
				ti.onShowWindow()
			}
		}
	})

	return ti, nil
}

// Dispose cleans up resources.
func (ti *TrayIcon) Dispose() {
	if ti.ni != nil {
		ti.ni.Dispose()
	}
}

// SetVisible shows or hides the tray icon.
func (ti *TrayIcon) SetVisible(visible bool) {
	if ti.ni != nil {
		ti.ni.SetVisible(visible)
	}
}

// SetLoggedIn updates the login state (affects icon color).
func (ti *TrayIcon) SetLoggedIn(loggedIn bool) {
	ti.loggedIn = loggedIn
	ti.updateIcon()
}

// SetConnectionStatus updates the connection state (affects icon color).
func (ti *TrayIcon) SetConnectionStatus(connected bool) {
	ti.connected = connected
	ti.updateIcon()
}

func (ti *TrayIcon) updateIcon() {
	if ti.ni == nil {
		return
	}
	var icon *walk.Icon
	if !ti.loggedIn {
		icon = ti.iconLost
	} else if !ti.connected {
		icon = ti.iconLost
	} else {
		icon = ti.iconConnected
	}
	ti.ni.SetIcon(icon)

	if ti.connected && ti.loggedIn {
		ti.ni.SetToolTip("Campus Auto Login — Connected")
	} else {
		ti.ni.SetToolTip("Campus Auto Login — Disconnected")
	}
}

// ShowBalloon displays a notification balloon.
func (ti *TrayIcon) ShowBalloon(title, message string) {
	if ti.ni != nil {
		ti.ni.ShowInfo(title, message)
	}
}

// =============================================================================
// Callback setters
// =============================================================================

func (ti *TrayIcon) OnShowWindow(fn func()) { ti.onShowWindow = fn }
func (ti *TrayIcon) OnReconnect(fn func())  { ti.onReconnect = fn }
func (ti *TrayIcon) OnOpenLogDir(fn func()) { ti.onOpenLogDir = fn }
func (ti *TrayIcon) OnQuit(fn func())       { ti.onQuit = fn }

// =============================================================================
// Context Menu (built directly on the NotifyIcon)
// =============================================================================

func (ti *TrayIcon) buildMenu() {
	// Show/Hide
	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏窗口")
	showAction.Triggered().Attach(func() {
		if ti.onShowWindow != nil {
			ti.onShowWindow()
		}
	})
	ti.ni.ContextMenu().Actions().Add(showAction)

	// Reconnect
	reconnectAction := walk.NewAction()
	reconnectAction.SetText("重新连接")
	reconnectAction.Triggered().Attach(func() {
		if ti.onReconnect != nil {
			ti.onReconnect()
		}
	})
	ti.ni.ContextMenu().Actions().Add(reconnectAction)

	// Open Log Dir
	openLogAction := walk.NewAction()
	openLogAction.SetText("打开日志目录")
	openLogAction.Triggered().Attach(func() {
		if ti.onOpenLogDir != nil {
			ti.onOpenLogDir()
		}
	})
	ti.ni.ContextMenu().Actions().Add(openLogAction)

	// Separator
	ti.ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	// Quit
	quitAction := walk.NewAction()
	quitAction.SetText("退出")
	quitAction.Triggered().Attach(func() {
		if ti.onQuit != nil {
			ti.onQuit()
		}
	})
	ti.ni.ContextMenu().Actions().Add(quitAction)
}

// =============================================================================
// Icon Generation — multi-resolution, anti-aliased .ico files at runtime
// =============================================================================
//
// Generates a true-color 32-bit ICO with multiple embedded sizes so Windows
// picks the sharpest match for the current DPI.  Each size is rendered with
// 4× supersampling for smooth anti-aliased edges and a visible border ring.
//
// Sizes: 16, 24, 32, 48, 64, 128, 256 (px)

func generateColorIcon(r, g, b byte) (*walk.Icon, error) {
	tmpDir := os.TempDir()
	icoFile := filepath.Join(tmpDir,
		fmt.Sprintf("campus_ico_v2_%02X%02X%02X.ico", r, g, b))

	if _, err := os.Stat(icoFile); os.IsNotExist(err) {
		icoData := buildICOData(r, g, b)
		if err := os.WriteFile(icoFile, icoData, 0644); err != nil {
			return nil, err
		}
	}

	return walk.NewIconFromFile(icoFile)
}

func buildICOData(r, g, b byte) []byte {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	const ss = 4 // supersample factor

	// --- Render each size at supersampled resolution then downsample ---
	type imgBlock struct {
		data   []byte
		offset int
	}
	var blocks []imgBlock

	icoHeaderSize := 6
	icoEntrySize := 16
	totalHeaderSize := icoHeaderSize + icoEntrySize*len(sizes)
	currentOffset := totalHeaderSize

	for _, sz := range sizes {
		bmpData := renderIconBMP(sz, ss, r, g, b)
		blocks = append(blocks, imgBlock{data: bmpData, offset: currentOffset})
		currentOffset += len(bmpData)
	}

	totalSize := currentOffset
	buf := make([]byte, totalSize)
	pos := 0

	// --- ICO Header ---
	binary.LittleEndian.PutUint16(buf[pos:], 0)                 // reserved
	binary.LittleEndian.PutUint16(buf[pos+2:], 1)               // type: ICO
	binary.LittleEndian.PutUint16(buf[pos+4:], uint16(len(sizes))) // count
	pos += 6

	// --- ICO Entries (one per size) ---
	for _, sz := range sizes {
		w, h := byte(sz), byte(sz)
		if sz >= 256 {
			w, h = 0, 0 // ICO spec: 256 → 0
		}
		buf[pos] = w
		pos++
		buf[pos] = h
		pos++
		buf[pos] = 0 // color palette
		pos++
		buf[pos] = 0 // reserved
		pos++
		binary.LittleEndian.PutUint16(buf[pos:], 1)  // planes
		pos += 2
		binary.LittleEndian.PutUint16(buf[pos:], 32) // bpp
		pos += 2
		// Image size and offset filled below (need block length)
		pos += 8 // placeholder
	}

	// Fill in image sizes + offsets
	pos = icoHeaderSize
	for i := range sizes {
		pos += 4 // skip w, h, palette, reserved
		pos += 4 // skip planes, bpp
		binary.LittleEndian.PutUint32(buf[pos:], uint32(len(blocks[i].data)))
		pos += 4
		binary.LittleEndian.PutUint32(buf[pos:], uint32(blocks[i].offset))
		pos += 4
	}

	// --- BMP Data Blocks ---
	for _, blk := range blocks {
		copy(buf[blk.offset:], blk.data)
	}

	return buf
}

// renderIconBMP renders a single BMP (DIB header + XOR pixels + AND mask)
// at the requested logical size, using ss× supersampling for anti-aliasing.
func renderIconBMP(size, ss int, r, g, b byte) []byte {
	renderW := size * ss
	renderH := size * ss

	// --- BMP layout ---
	bmpHeaderSize := 40
	xorRowSize := size * 4
	xorSize := xorRowSize * size
	andRowSize := ((size + 31) / 32) * 4
	andSize := andRowSize * size
	pixelDataSize := xorSize + andSize
	bmpDataSize := bmpHeaderSize + pixelDataSize

	bmp := make([]byte, bmpDataSize)
	pos := 0

	// DIB InfoHeader
	binary.LittleEndian.PutUint32(bmp[pos:], 40)
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], uint32(size))
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], uint32(size*2)) // biHeight: XOR + AND
	pos += 4
	binary.LittleEndian.PutUint16(bmp[pos:], 1) // planes
	pos += 2
	binary.LittleEndian.PutUint16(bmp[pos:], 32) // bpp
	pos += 2
	binary.LittleEndian.PutUint32(bmp[pos:], 0) // BI_RGB
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], uint32(pixelDataSize))
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], 2835) // X pixels/meter
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], 2835) // Y pixels/meter
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], 0) // clr used
	pos += 4
	binary.LittleEndian.PutUint32(bmp[pos:], 0) // clr important
	pos += 4

	// --- Supersampled render ---
	// Circle geometry (in supersampled coordinates)
	cx := float64(renderW) / 2.0
	cy := float64(renderH) / 2.0
	outerR := float64(size/2-1) * float64(ss)   // outer edge (leave 1px margin)
	borderR := outerR - float64(ss)*1.6          // border ring inner edge (~1.6 px thick)

	// Pre-render sup ersampled buffer: each pixel is {r, g, b, a} premultiplied
	type pixel struct{ r, g, b, a float64 }
	ssBuf := make([]pixel, renderW*renderH)

	// Dark border color (~35% brightness of base, but saturated)
	borderRF := float64(r) * 0.30
	borderGF := float64(g) * 0.30
	borderBF := float64(b) * 0.30

	// Base fill color
	fillRF := float64(r)
	fillGF := float64(g)
	fillBF := float64(b)

	for sy := 0; sy < renderH; sy++ {
		for sx := 0; sx < renderW; sx++ {
			dx := float64(sx) - cx + 0.5
			dy := float64(sy) - cy + 0.5
			dist := dx*dx + dy*dy
			// Avoid sqrt in inner loop — compare squared distances
			outerR2 := outerR * outerR
			borderR2 := borderR * borderR

			idx := sy*renderW + sx

			switch {
			case dist <= borderR2:
				// Inner fill — base color, fully opaque
				ssBuf[idx] = pixel{fillRF, fillGF, fillBF, 255.0}
			case dist <= outerR2:
				// Border ring — dark band
				ssBuf[idx] = pixel{borderRF, borderGF, borderBF, 255.0}
			default:
				// Outside — compute anti-aliased edge
				// How far past outerR (in subpixel units)?
				d := dist - outerR2
				// One supersample-pixel-wide soft edge
				softEdge := float64(ss) * outerR * 2.0
				if d < softEdge {
					alpha := 255.0 * (1.0 - d/softEdge)
					if alpha < 0 {
						alpha = 0
					}
					ssBuf[idx] = pixel{borderRF, borderGF, borderBF, alpha}
				} else {
					ssBuf[idx] = pixel{0, 0, 0, 0}
				}
			}
		}
	}

	// --- Downsample to target resolution (box filter) ---
	area := float64(ss * ss)
	for by := 0; by < size; by++ { // BMP rows: bottom-up
		imgY := size - 1 - by
		rowOff := pos + by*xorRowSize
		for x := 0; x < size; x++ {
			px := rowOff + x*4

			var sumR, sumG, sumB, sumA float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					p := ssBuf[(imgY*ss+sy)*renderW + (x*ss + sx)]
					sumR += p.r * p.a // un-premultiply later
					sumG += p.g * p.a
					sumB += p.b * p.a
					sumA += p.a
				}
			}

			if sumA > 0.5 {
				// Reconstruct straight alpha
				bmp[px] = byte(clamp(sumB/sumA+0.5, 0, 255))   // B
				bmp[px+1] = byte(clamp(sumG/sumA+0.5, 0, 255)) // G
				bmp[px+2] = byte(clamp(sumR/sumA+0.5, 0, 255)) // R
				bmp[px+3] = byte(clamp(sumA/area+0.5, 0, 255)) // A
			}
			// else: already zero (transparent)
		}
	}

	// AND mask already zero-initialized at end of BMP buffer

	return bmp
}

func clamp(v float64, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
