package main

import (
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ClassicIdenticon with 100% deterministic, bit-perfect design + 2-color mode
type ClassicIdenticon struct {
	source []byte
	size   int
}

// NewClassicIdenticon creates a generator with classic look
func NewClassicIdenticon(source []byte) *ClassicIdenticon {
	return &ClassicIdenticon{
		source: source,
		size:   256,
	}
}

// mapValue maps a value from one range to another
func mapValue(value uint32, vmin, vmax, dmin, dmax uint32) float32 {
	if vmax == vmin {
		return float32(dmin)
	}
	return float32(dmin) + float32(value-vmin)*float32(dmax-dmin)/float32(vmax-vmin)
}

// getBit returns the n-th bit (0-indexed) from source
func (identicon *ClassicIdenticon) getBit(n int) bool {
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
func (identicon *ClassicIdenticon) getByte(n int) byte {
	if len(identicon.source) == 0 {
		return 0
	}
	return identicon.source[n%len(identicon.source)]
}

// foreground computes primary color
func (identicon *ClassicIdenticon) foreground() color.Color {
	if len(identicon.source) < 32 {
		return color.RGBA{0, 0, 0, 255}
	}

	// Use bit 255 to decide: 0 → original HSL, 1 → palette
	if !identicon.getBit(255) {
		// Original HSL algorithm — soft and harmonious
		h1 := (uint16(identicon.getByte(28)) & 0x0f) << 8
		h2 := uint16(identicon.getByte(29))
		h := uint32(h1 | h2)
		s := uint32(identicon.getByte(30))
		l := uint32(identicon.getByte(31))

		hue := mapValue(h, 0, 4095, 0, 360)
		sat := mapValue(s, 0, 255, 0, 20)
		lum := mapValue(l, 0, 255, 0, 20)

		return identicon.hslToRgb(hue, 65.0-sat, 75.0-lum)
	}

	// Vibrant color palette — 16 beautiful, distinct colors
	palette := []color.RGBA{
		{0x00, 0xbf, 0x93, 0xff}, // turquoise
		{0x2d, 0xcc, 0x70, 0xff}, // mint
		{0x42, 0xe4, 0x53, 0xff}, // green
		{0xf1, 0xc4, 0x0f, 0xff}, // yellowOrange
		{0xe6, 0x7f, 0x22, 0xff}, // brown
		{0xff, 0x94, 0x4e, 0xff}, // orange
		{0xe8, 0x4c, 0x3d, 0xff}, // red
		{0x35, 0x98, 0xdb, 0xff}, // blue
		{0x9a, 0x59, 0xb5, 0xff}, // purple
		{0xef, 0x3e, 0x96, 0xff}, // magenta
		{0xdf, 0x21, 0xb9, 0xff}, // violet
		{0x7d, 0xc2, 0xd2, 0xff}, // lightBlue
		{0x16, 0xa0, 0x86, 0xff}, // turquoiseIntense
		{0x27, 0xae, 0x61, 0xff}, // mintIntense
		{0x24, 0xc3, 0x33, 0xff}, // greenIntense
		{0x1c, 0xab, 0xbb, 0xff}, // lightBlueIntense
	}

	// Use bits 248-251 to select color (4 bits → 16 colors)
	colorIndex := 0
	for i := 0; i < 4; i++ {
		if identicon.getBit(248 + i) {
			colorIndex |= 1 << i
		}
	}
	return palette[colorIndex%len(palette)]
}

// secondaryColor computes second color (for 2-color mode)
func (identicon *ClassicIdenticon) secondaryColor() color.Color {
	if len(identicon.source) < 32 {
		return color.RGBA{100, 100, 100, 255}
	}

	// Use different bits: 244-247 for second color
	colorIndex := 0
	for i := 0; i < 4; i++ {
		if identicon.getBit(244 + i) {
			colorIndex |= 1 << i
		}
	}

	palette := []color.RGBA{
		{0x34, 0x49, 0x5e, 0xff}, // darkBlue
		{0x95, 0xa5, 0xa5, 0xff}, // grey
		{0xd2, 0x54, 0x00, 0xff}, // brownIntense
		{0xc1, 0x39, 0x2b, 0xff}, // redIntense
		{0x29, 0x7f, 0xb8, 0xff}, // blueIntense
		{0x8d, 0x44, 0xad, 0xff}, // purpleIntense
		{0xbe, 0x12, 0x7e, 0xff}, // violetIntense
		{0xe5, 0x23, 0x83, 0xff}, // magentaIntense
		{0x27, 0xae, 0x61, 0xff}, // mintIntense
		{0x24, 0xc3, 0x33, 0xff}, // greenIntense
		{0xd9, 0xd9, 0x21, 0xff}, // yellowIntense
		{0xf3, 0x9c, 0x11, 0xff}, // yellowOrangeIntense
		{0xff, 0x55, 0x00, 0xff}, // orangeIntense
		{0x1c, 0xab, 0xbb, 0xff}, // lightBlueIntense
		{0x23, 0x23, 0x23, 0xff}, // lightBlackIntense
		{0x7e, 0x8c, 0x8d, 0xff}, // greyIntense
	}

	return palette[colorIndex%len(palette)]
}

// hslToRgb converts HSL to RGB in original style
func (identicon *ClassicIdenticon) hslToRgb(h, s, l float32) color.Color {
	hue := h / 360.0
	sat := s / 100.0
	lum := l / 100.0

	var b float32
	if lum <= 0.5 {
		b = lum * (sat + 1.0)
	} else {
		b = lum + sat - lum*sat
	}
	a := lum*2.0 - b

	red := identicon.hueToRgb(a, b, hue+1.0/3.0)
	green := identicon.hueToRgb(a, b, hue)
	blue := identicon.hueToRgb(a, b, hue-1.0/3.0)

	return color.RGBA{
		R: uint8(math.Round(float64(red * 255.0))),
		G: uint8(math.Round(float64(green * 255.0))),
		B: uint8(math.Round(float64(blue * 255.0))),
		A: 255,
	}
}

// hueToRgb helper for color conversion
func (identicon *ClassicIdenticon) hueToRgb(a, b, hue float32) float32 {
	if hue < 0 {
		hue += 1.0
	} else if hue >= 1.0 {
		hue -= 1.0
	}

	switch {
	case hue < 1.0/6.0:
		return a + (b-a)*6.0*hue
	case hue < 0.5:
		return b
	case hue < 2.0/3.0:
		return a + (b-a)*(2.0/3.0-hue)*6.0
	default:
		return a
	}
}

// drawRect draws a solid rectangle
func (identicon *ClassicIdenticon) drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	rect := img.Bounds()
	x0 = max(x0, rect.Min.X)
	y0 = max(y0, rect.Min.Y)
	x1 = min(x1, rect.Max.X)
	y1 = min(y1, rect.Max.Y)

	if x0 >= x1 || y0 >= y1 {
		return
	}

	r, g, b, a := c.RGBA()
	rgba := color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}

	for y := y0; y < y1; y++ {
		rowStart := img.PixOffset(x0, y)
		for x := 0; x < x1-x0; x++ {
			idx := rowStart + x*4
			img.Pix[idx] = rgba.R
			img.Pix[idx+1] = rgba.G
			img.Pix[idx+2] = rgba.B
			img.Pix[idx+3] = rgba.A
		}
	}
}

// generatePixelPattern generates 5x5 symmetric pixel grid — using individual bits
// Returns two layers: primary and secondary
func (identicon *ClassicIdenticon) generatePixelPattern() ([]bool, []bool) {
	primary := make([]bool, 25)
	secondary := make([]bool, 25)

	// Use bits 0-14 for primary pattern (15 bits)
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

	// Use bits 15-29 for secondary pattern (next 15 bits)
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

// Generate creates the identicon for UI display (respects theme)
func (identicon *ClassicIdenticon) Generate() image.Image {
	const (
		pixelSize  = 36
		spriteSize = 5
		margin     = (256 - pixelSize*spriteSize) / 2
	)

	primaryColor := identicon.foreground()
	secondaryColor := identicon.secondaryColor()
	img := image.NewRGBA(image.Rect(0, 0, identicon.size, identicon.size))

	// Background adapts to theme — use bits 252-254 to pick variation
	bgChoice := 0
	for i := 0; i < 3; i++ {
		if identicon.getBit(252 + i) {
			bgChoice |= 1 << i
		}
	}
	bgChoice %= 3

	lightBackgrounds := []color.RGBA{
		{255, 255, 255, 255}, // pure white
		{243, 245, 247, 255}, // light1
		{236, 240, 241, 255}, // light2
	}
	darkBackgrounds := []color.RGBA{
		{30, 30, 30, 255},    // dark gray
		{45, 62, 80, 255},     // darkBlueIntense
		{57, 57, 57, 255},     // dark2
	}

	var bg color.RGBA
	if fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark {
		bg = darkBackgrounds[bgChoice]
	} else {
		bg = lightBackgrounds[bgChoice]
	}

	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = bg.R
		img.Pix[i+1] = bg.G
		img.Pix[i+2] = bg.B
		img.Pix[i+3] = bg.A
	}

	primaryPixels, secondaryPixels := identicon.generatePixelPattern()

	// Draw secondary pixels first (background layer)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if secondaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				identicon.drawRect(img, x, y, x+pixelSize, y+pixelSize, secondaryColor)
			}
		}
	}

	// Draw primary pixels on top (foreground layer)
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if primaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				identicon.drawRect(img, x, y, x+pixelSize, y+pixelSize, primaryColor)
			}
		}
	}

	return img
}

// GenerateForExport generates identicon with fixed background for saving
func (identicon *ClassicIdenticon) GenerateForExport(transparent bool) image.Image {
	const (
		pixelSize  = 36
		spriteSize = 5
		margin     = (256 - pixelSize*spriteSize) / 2
	)

	primaryColor := identicon.foreground()
	secondaryColor := identicon.secondaryColor()
	img := image.NewRGBA(image.Rect(0, 0, identicon.size, identicon.size))

	// Set export background
	var bg color.RGBA
	if transparent {
		bg = color.RGBA{0, 0, 0, 0} // fully transparent
	} else {
		// Use bits 252-254 for background choice
		bgChoice := 0
		for i := 0; i < 3; i++ {
			if identicon.getBit(252 + i) {
				bgChoice |= 1 << i
			}
		}
		bgChoice %= 3

		lightBackgrounds := []color.RGBA{
			{255, 255, 255, 255},
			{243, 245, 247, 255},
			{236, 240, 241, 255},
		}
		bg = lightBackgrounds[bgChoice]
	}

	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = bg.R
		img.Pix[i+1] = bg.G
		img.Pix[i+2] = bg.B
		img.Pix[i+3] = bg.A
	}

	primaryPixels, secondaryPixels := identicon.generatePixelPattern()

	// Draw secondary pixels first
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if secondaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				identicon.drawRect(img, x, y, x+pixelSize, y+pixelSize, secondaryColor)
			}
		}
	}

	// Draw primary pixels on top
	for row := 0; row < spriteSize; row++ {
		for col := 0; col < spriteSize; col++ {
			if primaryPixels[row*spriteSize+col] {
				x := col*pixelSize + margin
				y := row*pixelSize + margin
				identicon.drawRect(img, x, y, x+pixelSize, y+pixelSize, primaryColor)
			}
		}
	}

	return img
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

	generateBtn := widget.NewButton("Generate Identicon", func() {
		if input.Text == "" {
			dialog.ShowInformation("Error", "Please enter some text.", myWindow)
			return
		}

		hash := sha256.Sum256([]byte(input.Text))

		// Generate for UI (theme-adaptive background)
		identiconDisplay := NewClassicIdenticon(hash[:])
		imgDisplay := identiconDisplay.Generate()

		fyneImg := canvas.NewImageFromImage(imgDisplay)
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

		// Save button
		saveBtn := widget.NewButton("Save as PNG", func() {
			hashForSave := sha256.Sum256([]byte(input.Text))
			identiconForSave := NewClassicIdenticon(hashForSave[:])
			imgToSave := identiconForSave.GenerateForExport(transparentCheck.Checked)

			fileDialog := dialog.NewFileSave(
				func(uc fyne.URIWriteCloser, err error) {
					if err != nil || uc == nil {
						return
					}
					defer uc.Close()

					err = png.Encode(uc, imgToSave)
					if err != nil {
						dialog.ShowError(err, myWindow)
						return
					}

					bgMsg := "white background"
					if transparentCheck.Checked {
						bgMsg = "transparent background"
					}
					dialog.ShowInformation("Success", "Image saved with "+bgMsg+"!", myWindow)
				},
				myWindow,
			)
			fileDialog.SetFileName("identicon.png")
			fileDialog.Show()
		})

		imageContainer := container.NewCenter(fyneImg)

		content := container.NewVBox(
			imageContainer,

			layout.NewSpacer(),

			container.NewHBox(
				layout.NewSpacer(),
				saveBtn,
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
		// Refresh window content to apply theme
		myWindow.Content().Refresh()
	})
	themeSwitch.Importance = widget.LowImportance

	// Create top-right aligned layout — with your exact style
	topBar := container.NewHBox(
		layout.NewSpacer(), // pushes toggle to the right
		themeSwitch,
	)

	content := container.NewBorder(
		topBar,           // top: theme switch right-aligned
		nil,              // bottom: nothing
		nil,              // left: nothing
		nil,              // right: nothing
		container.NewVBox( // center: main content
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
