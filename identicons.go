package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// OptimizedIdenticon with indexed colors for smaller file sizes
type OptimizedIdenticon struct {
	source []byte
	size   int
}

// NewOptimizedIdenticon creates a generator with indexed colors
func NewOptimizedIdenticon(source []byte) *OptimizedIdenticon {
	return &OptimizedIdenticon{
		source: source,
		size:   256,
	}
}

// NewOptimizedIdenticonWithSize creates a generator with custom size
func NewOptimizedIdenticonWithSize(source []byte, size int) *OptimizedIdenticon {
	return &OptimizedIdenticon{
		source: source,
		size:   size,
	}
}

// getBit returns the n-th bit (0-indexed) from source
func (identicon *OptimizedIdenticon) getBit(n int) bool {
	if len(identicon.source) == 0 || n < 0 {
		return false
	}
	byteIndex := n / 8
	bitIndex := n % 8
	if byteIndex >= len(identicon.source) {
		return false
	}
	return (identicon.source[byteIndex]>>bitIndex)&1 == 1
}

// getByte returns the n-th byte, wraps around if needed
func (identicon *OptimizedIdenticon) getByte(n int) byte {
	if len(identicon.source) == 0 {
		return 0
	}
	return identicon.source[n%len(identicon.source)]
}

// getColorIndices returns color indices for indexed version
func (identicon *OptimizedIdenticon) getColorIndices() (primaryIndex, secondaryIndex, bgIndex uint8) {
	if len(identicon.source) < 32 {
		return 0, 1, 2
	}

	// Primary color index (4 bits → 16 colors)
	primaryIndex = 0
	for i := 0; i < 4; i++ {
		if identicon.getBit(248 + i) {
			primaryIndex |= 1 << i
		}
	}
	primaryIndex %= 16

	// Secondary color index (4 bits → 16 colors)
	secondaryIndex = 0
	for i := 0; i < 4; i++ {
		if identicon.getBit(244 + i) {
			secondaryIndex |= 1 << i
		}
	}
	secondaryIndex %= 16

	// Background choice (2 bits → 4 options)
	bgChoice := 0
	for i := 0; i < 2; i++ {
		if identicon.getBit(252 + i) {
			bgChoice |= 1 << i
		}
	}
	bgIndex = uint8(bgChoice % 3) // 0, 1, or 2

	return primaryIndex, secondaryIndex, bgIndex
}

// generatePixelPattern generates 5x5 symmetric pixel grid
func (identicon *OptimizedIdenticon) generatePixelPattern() ([]bool, []bool) {
	primary := make([]bool, 25)
	secondary := make([]bool, 25)

	// Use bits 0-14 for primary pattern
	bitIndex := 0
	for row := 0; row < 5; row++ {
		for col := 0; col < 3; col++ {
			paint := identicon.getBit(bitIndex)
			bitIndex++

			ix := row*5 + col
			mirrorIx := row*5 + (4 - col)
			primary[ix] = paint
			primary[mirrorIx] = paint
		}
	}

	// Use bits 15-29 for secondary pattern
	for row := 0; row < 5; row++ {
		for col := 0; col < 3; col++ {
			paint := identicon.getBit(bitIndex)
			bitIndex++

			ix := row*5 + col
			mirrorIx := row*5 + (4 - col)
			secondary[ix] = paint
			secondary[mirrorIx] = paint
		}
	}

	return primary, secondary
}

// createPalette creates an optimized palette with only necessary colors
func createPalette(primaryIdx, secondaryIdx, bgIdx uint8, darkMode bool) color.Palette {
	// Optimized color palettes - only 16 colors per palette
	primaryPalette := []color.Color{
		color.RGBA{0x00, 0xbf, 0x93, 0xff}, // turquoise
		color.RGBA{0x2d, 0xcc, 0x70, 0xff}, // mint
		color.RGBA{0x42, 0xe4, 0x53, 0xff}, // green
		color.RGBA{0xf1, 0xc4, 0x0f, 0xff}, // yellowOrange
		color.RGBA{0xe6, 0x7f, 0x22, 0xff}, // brown
		color.RGBA{0xff, 0x94, 0x4e, 0xff}, // orange
		color.RGBA{0xe8, 0x4c, 0x3d, 0xff}, // red
		color.RGBA{0x35, 0x98, 0xdb, 0xff}, // blue
		color.RGBA{0x9a, 0x59, 0xb5, 0xff}, // purple
		color.RGBA{0xef, 0x3e, 0x96, 0xff}, // magenta
		color.RGBA{0xdf, 0x21, 0xb9, 0xff}, // violet
		color.RGBA{0x7d, 0xc2, 0xd2, 0xff}, // lightBlue
		color.RGBA{0x16, 0xa0, 0x86, 0xff}, // turquoiseIntense
		color.RGBA{0x27, 0xae, 0x61, 0xff}, // mintIntense
		color.RGBA{0x24, 0xc3, 0x33, 0xff}, // greenIntense
		color.RGBA{0x1c, 0xab, 0xbb, 0xff}, // lightBlueIntense
	}

	secondaryPalette := []color.Color{
		color.RGBA{0x34, 0x49, 0x5e, 0xff}, // darkBlue
		color.RGBA{0x95, 0xa5, 0xa5, 0xff}, // grey
		color.RGBA{0xd2, 0x54, 0x00, 0xff}, // brownIntense
		color.RGBA{0xc1, 0x39, 0x2b, 0xff}, // redIntense
		color.RGBA{0x29, 0x7f, 0xb8, 0xff}, // blueIntense
		color.RGBA{0x8d, 0x44, 0xad, 0xff}, // purpleIntense
		color.RGBA{0xbe, 0x12, 0x7e, 0xff}, // violetIntense
		color.RGBA{0xe5, 0x23, 0x83, 0xff}, // magentaIntense
		color.RGBA{0x27, 0xae, 0x61, 0xff}, // mintIntense
		color.RGBA{0x24, 0xc3, 0x33, 0xff}, // greenIntense
		color.RGBA{0xd9, 0xd9, 0x21, 0xff}, // yellowIntense
		color.RGBA{0xf3, 0x9c, 0x11, 0xff}, // yellowOrangeIntense
		color.RGBA{0xff, 0x55, 0x00, 0xff}, // orangeIntense
		color.RGBA{0x1c, 0xab, 0xbb, 0xff}, // lightBlueIntense
		color.RGBA{0x23, 0x23, 0x23, 0xff}, // lightBlackIntense
		color.RGBA{0x7e, 0x8c, 0x8d, 0xff}, // greyIntense
	}

	// Background colors based on mode
	lightBackgrounds := []color.Color{
		color.RGBA{255, 255, 255, 255},   // white
		color.RGBA{243, 245, 247, 255},   // light gray 1
		color.RGBA{236, 240, 241, 255},   // light gray 2
		color.RGBA{0, 0, 0, 0},           // transparent (position 3)
	}
	
	darkBackgrounds := []color.Color{
		color.RGBA{30, 30, 30, 255},      // dark gray
		color.RGBA{45, 62, 80, 255},      // dark blue
		color.RGBA{57, 57, 57, 255},      // dark gray 2
		color.RGBA{0, 0, 0, 0},           // transparent (position 3)
	}

	// Palette in exact order:
	// 0: Background
	// 1: Primary color
	// 2: Secondary color
	// 3: Transparent (optional)
	palette := make(color.Palette, 0, 4)
	
	// Background first
	if darkMode {
		palette = append(palette, darkBackgrounds[bgIdx])
	} else {
		palette = append(palette, lightBackgrounds[bgIdx])
	}
	
	// Then primary and secondary colors
	palette = append(palette, 
		primaryPalette[primaryIdx],
		secondaryPalette[secondaryIdx],
		color.RGBA{0, 0, 0, 0}, // transparent as last option
	)
	
	return palette
}

// GenerateIndexed creates indexed image for display
func (identicon *OptimizedIdenticon) GenerateIndexed(darkMode bool) *image.Paletted {
	const (
		spriteSize = 5
	)
	
	pixelSize := identicon.size / 8
	margin := (identicon.size - pixelSize*spriteSize) / 2

	// Determine color indices
	primaryIdx, secondaryIdx, bgIdx := identicon.getColorIndices()
	
	// Create palette
	palette := createPalette(primaryIdx, secondaryIdx, bgIdx, darkMode)
	
	// Create indexed image
	img := image.NewPaletted(image.Rect(0, 0, identicon.size, identicon.size), palette)
	
	// Fill entire background with index 0 (background color)
	for i := 0; i < identicon.size; i++ {
		for j := 0; j < identicon.size; j++ {
			img.SetColorIndex(j, i, 0)
		}
	}

	primaryPixels, secondaryPixels := identicon.generatePixelPattern()

	// Draw secondary pixels first (index 2)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if secondaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				// Fill rectangle with index 2
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < identicon.size && py < identicon.size {
							img.SetColorIndex(px, py, 2)
						}
					}
				}
			}
		}
	}

	// Draw primary pixels on top (index 1)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if primaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				// Fill rectangle with index 1
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < identicon.size && py < identicon.size {
							img.SetColorIndex(px, py, 1)
						}
					}
				}
			}
		}
	}

	return img
}

// GenerateForExportOptimized for indexed export
func (identicon *OptimizedIdenticon) GenerateForExportOptimized(transparent bool) *image.Paletted {
	const (
		spriteSize = 5
	)
	
	pixelSize := identicon.size / 8
	margin := (identicon.size - pixelSize*spriteSize) / 2

	// Determine color indices
	primaryIdx, secondaryIdx, bgIdx := identicon.getColorIndices()
	
	// For transparent background, set bgIdx to 3 (transparent)
	if transparent {
		bgIdx = 3
	}
	
	// Palette for export (always light mode for better compatibility)
	palette := createPalette(primaryIdx, secondaryIdx, bgIdx, false)
	
	// Create image
	img := image.NewPaletted(image.Rect(0, 0, identicon.size, identicon.size), palette)
	
	// Fill background
	bgIndex := uint8(0)
	if transparent {
		bgIndex = 3 // transparent
	}
	
	for i := 0; i < identicon.size; i++ {
		for j := 0; j < identicon.size; j++ {
			img.SetColorIndex(j, i, bgIndex)
		}
	}

	primaryPixels, secondaryPixels := identicon.generatePixelPattern()

	// Secondary pixels (index 2)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if secondaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < identicon.size && py < identicon.size {
							img.SetColorIndex(px, py, 2)
						}
					}
				}
			}
		}
	}

	// Primary pixels (index 1)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if primaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < identicon.size && py < identicon.size {
							img.SetColorIndex(px, py, 1)
						}
					}
				}
			}
		}
	}

	return img
}

// Generate48x48ForFace creates a 48x48 pixel image specifically for Face headers
func (identicon *OptimizedIdenticon) Generate48x48ForFace(transparent bool) *image.Paletted {
	const (
		size       = 48
		spriteSize = 5
		pixelSize  = 6 // 48/8 = 6
		margin     = (size - pixelSize*spriteSize) / 2 // = 9
	)

	// Determine color indices
	primaryIdx, secondaryIdx, bgIdx := identicon.getColorIndices()
	
	// For transparent background, set bgIdx to 3 (transparent)
	if transparent {
		bgIdx = 3
	}
	
	// Palette for export (always light mode for better compatibility)
	palette := createPalette(primaryIdx, secondaryIdx, bgIdx, false)
	
	// Create image
	img := image.NewPaletted(image.Rect(0, 0, size, size), palette)
	
	// Fill background
	bgIndex := uint8(0)
	if transparent {
		bgIndex = 3 // transparent
	}
	
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			img.SetColorIndex(j, i, bgIndex)
		}
	}

	primaryPixels, secondaryPixels := identicon.generatePixelPattern()

	// Secondary pixels (index 2)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if secondaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < size && py < size {
							img.SetColorIndex(px, py, 2)
						}
					}
				}
			}
		}
	}

	// Primary pixels (index 1)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if primaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				for py := y; py < y+pixelSize; py++ {
					for px := x; px < x+pixelSize; px++ {
						if px < size && py < size {
							img.SetColorIndex(px, py, 1)
						}
					}
				}
			}
		}
	}

	return img
}

// Helper function to write Face header file
func writeFaceFile(filename, b64 string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString("Face:"); err != nil {
		return err
	}

	lenB64 := len(b64)
	if lenB64 > 72 {
		if _, err := file.WriteString(" " + b64[:70]); err != nil {
			return err
		}
		pos := 70
		for pos < lenB64 {
			remaining := lenB64 - pos
			chunk := 75
			if remaining < 75 {
				chunk = remaining
			}
			if _, err := file.WriteString("\n " + b64[pos:pos+chunk]); err != nil {
				return err
			}
			pos += chunk
		}
	} else {
		if _, err := file.WriteString(" " + b64); err != nil {
			return err
		}
	}
	
	_, err = file.WriteString("\n")
	return err
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Identicons")
	myWindow.Resize(fyne.NewSize(480, 480))

	isDarkTheme := true
	myApp.Settings().SetTheme(theme.DarkTheme())

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter text...")

	generateBtn := widget.NewButton("Generate", func() {
		if input.Text == "" {
			dialog.ShowInformation("Error", "Please enter some text.", myWindow)
			return
		}

		hash := sha256.Sum256([]byte(input.Text))

		// Optimized version for UI
		identicon := NewOptimizedIdenticon(hash[:])
		imgDisplay := identicon.GenerateIndexed(isDarkTheme)

		// Convert to RGBA for Fyne (Fyne requires RGBA)
		rgbaImg := image.NewRGBA(imgDisplay.Bounds())
		for y := 0; y < rgbaImg.Bounds().Dy(); y++ {
			for x := 0; x < rgbaImg.Bounds().Dx(); x++ {
				// Convert color index to actual color
				colorIndex := imgDisplay.ColorIndexAt(x, y)
				color := imgDisplay.Palette[colorIndex]
				rgbaImg.Set(x, y, color)
			}
		}

		fyneImg := canvas.NewImageFromImage(rgbaImg)
		fyneImg.FillMode = canvas.ImageFillContain
		fyneImg.SetMinSize(fyne.NewSize(256, 256))

		transparentCheck := widget.NewCheck("", nil)
		transparentCheck.SetChecked(true)

		var labelColor color.Color
		if isDarkTheme {
			labelColor = color.White
		} else {
			labelColor = color.Black
		}

		transparentLabel := canvas.NewText("Transparent Background (PNG)", labelColor)
		transparentLabel.TextSize = 14

		transparentToggleContainer := container.NewHBox(
			transparentCheck,
			transparentLabel,
			layout.NewSpacer(),
		)

		// Save buttons
		savePngBtn := widget.NewButton("Save as PNG", func() {
			hashForSave := sha256.Sum256([]byte(input.Text))
			identiconForSave := NewOptimizedIdenticon(hashForSave[:])
			imgToSave := identiconForSave.GenerateForExportOptimized(transparentCheck.Checked)

			fileDialog := dialog.NewFileSave(
				func(uc fyne.URIWriteCloser, err error) {
					if err != nil || uc == nil {
						return
					}
					defer uc.Close()

					// PNG encoder with optimizations
					encoder := png.Encoder{
						CompressionLevel: png.BestCompression,
					}
					
					err = encoder.Encode(uc, imgToSave)
					if err != nil {
						dialog.ShowError(err, myWindow)
						return
					}

					bgMsg := "white background"
					if transparentCheck.Checked {
						bgMsg = "transparent background"
					}
					
					dialog.ShowInformation("Success", 
						"Image saved with "+bgMsg+"!\n"+
						"Estimated size: ~2-5 KB", myWindow)
				},
				myWindow,
			)
			fileDialog.SetFileName("identicon.png")
			fileDialog.Show()
		})

		// New: Save as Face header button
		saveFaceBtn := widget.NewButton("Save as Face Header", func() {
			hashForSave := sha256.Sum256([]byte(input.Text))
			identiconForSave := NewOptimizedIdenticonWithSize(hashForSave[:], 48)
			
			// Generate 48x48 image
			imgToSave := identiconForSave.Generate48x48ForFace(transparentCheck.Checked)

			fileDialog := dialog.NewFileSave(
				func(uc fyne.URIWriteCloser, err error) {
					if err != nil || uc == nil {
						return
					}
					defer uc.Close()

					// Encode image to PNG
					var buf bytes.Buffer
					encoder := png.Encoder{
						CompressionLevel: png.BestCompression,
					}
					
					err = encoder.Encode(&buf, imgToSave)
					if err != nil {
						dialog.ShowError(err, myWindow)
						return
					}

					// Convert to base64
					b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

					// Write Face header format
					if err := writeFaceFile(uc.URI().Path(), b64); err != nil {
						dialog.ShowError(err, myWindow)
						return
					}

					bgMsg := "white background"
					if transparentCheck.Checked {
						bgMsg = "transparent background"
					}
					
					dialog.ShowInformation("Success", 
						"48x48 Face header saved with "+bgMsg+"!\n"+
						"Base64 size: "+fmt.Sprintf("%d", len(b64))+" bytes\n"+
						"File: "+uc.URI().Name(), myWindow)
				},
				myWindow,
			)
			fileDialog.SetFileName("face.txt")
			fileDialog.Show()
		})

		imageContainer := container.NewCenter(fyneImg)

		content := container.NewVBox(
			imageContainer,
			layout.NewSpacer(),
			container.NewHBox(
				layout.NewSpacer(),
				savePngBtn,
				widget.NewLabel("   "),
				saveFaceBtn,
				layout.NewSpacer(),
			),
			container.NewHBox(
				layout.NewSpacer(),
				transparentToggleContainer,
				layout.NewSpacer(),
			),
		)

		dialog.ShowCustom("", "OK", content, myWindow)
	})

	themeSwitch := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if isDarkTheme {
			myApp.Settings().SetTheme(theme.LightTheme())
			isDarkTheme = false
		} else {
			myApp.Settings().SetTheme(theme.DarkTheme())
			isDarkTheme = true
		}
		myWindow.Content().Refresh()
	})
	themeSwitch.Importance = widget.LowImportance

	topBar := container.NewHBox(
		layout.NewSpacer(),
		themeSwitch,
	)

	content := container.NewBorder(
		topBar,
		nil,
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			input,
			layout.NewSpacer(),
			container.NewHBox(
				layout.NewSpacer(),
				generateBtn,
				layout.NewSpacer(),
			),
			layout.NewSpacer(),
		),
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
