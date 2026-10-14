package bot

import (
	"image"
	"image/color"
	"math"
	"strings"
)

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }
func ellipse(img *image.RGBA, box image.Rectangle, fill, outline color.Color, width int) {
	cx, cy := float64(box.Min.X+box.Max.X)/2, float64(box.Min.Y+box.Max.Y)/2
	rx, ry := float64(box.Dx())/2, float64(box.Dy())/2
	for y := box.Min.Y; y <= box.Max.Y; y++ {
		for x := box.Min.X; x <= box.Max.X; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx/(rx*rx)+dy*dy/(ry*ry) <= 1 {
				c := fill
				if width > 0 && outline != nil && (dx*dx/math.Pow(rx-float64(width), 2)+dy*dy/math.Pow(ry-float64(width), 2) >= 1) {
					c = outline
				}
				if c != nil {
					img.Set(x, y, c)
				}
			}
		}
	}
}
func thickLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color, width int) {
	steps := max(abs(x1-x0), abs(y1-y0))
	if steps == 0 {
		return
	}
	for i := 0; i <= steps; i++ {
		fillCircle(img, x0+(x1-x0)*i/steps, y0+(y1-y0)*i/steps, width/2, c)
	}
}
func arc(img *image.RGBA, box image.Rectangle, start, end float64, c color.Color, width int) {
	cx, cy := float64(box.Min.X+box.Max.X)/2, float64(box.Min.Y+box.Max.Y)/2
	rx, ry := float64(box.Dx())/2, float64(box.Dy())/2
	for angle := start; angle <= end; angle += 0.3 {
		rad := angle * math.Pi / 180
		fillCircle(img, int(cx+rx*math.Cos(rad)), int(cy+ry*math.Sin(rad)), width/2, c)
	}
}
func borderedRect(img *image.RGBA, box image.Rectangle, fill, outline color.Color, width int) {
	if fill != nil {
		fillRect(img, box, fill)
	}
	fillRect(img, image.Rect(box.Min.X, box.Min.Y, box.Max.X, box.Min.Y+width), outline)
	fillRect(img, image.Rect(box.Min.X, box.Max.Y-width, box.Max.X, box.Max.Y), outline)
	fillRect(img, image.Rect(box.Min.X, box.Min.Y, box.Min.X+width, box.Max.Y), outline)
	fillRect(img, image.Rect(box.Max.X-width, box.Min.Y, box.Max.X, box.Max.Y), outline)
}
func memeText(img *image.RGBA, text string, y, size, stroke int, ink color.Color) {
	lines := wrapRunes(strings.Join(strings.Fields(text), " "), 15)
	if y < 0 {
		y = (650 - len(lines)*(size+8)) / 2
	}
	for _, line := range lines {
		x := (900 - textWidth(line, size)) / 2
		if stroke > 0 {
			for dx := -stroke; dx <= stroke; dx++ {
				for dy := -stroke; dy <= stroke; dy++ {
					drawTextSize(img, line, x+dx, y+size+dy, rgb(0, 0, 0), size)
				}
			}
		}
		drawTextSize(img, line, x, y+size, ink, size)
		y += size + 8
	}
}
func memeFace(img *image.RGBA, cx, cy, r int, shocked bool) {
	ellipse(img, image.Rect(cx-r, cy-r, cx+r, cy+r), rgb(245, 203, 157), rgb(50, 35, 30), 5)
	eyeY := cy - r/4
	if shocked {
		ellipse(img, image.Rect(cx-r/3, eyeY-12, cx-r/3+24, eyeY+15), rgb(25, 25, 25), nil, 0)
		ellipse(img, image.Rect(cx+r/3-24, eyeY-12, cx+r/3, eyeY+15), rgb(25, 25, 25), nil, 0)
		ellipse(img, image.Rect(cx-22, cy+r/5, cx+22, cy+r/2), rgb(50, 20, 20), nil, 0)
	} else {
		ellipse(img, image.Rect(cx-r/3, eyeY, cx-r/3+18, eyeY+18), rgb(35, 35, 35), nil, 0)
		ellipse(img, image.Rect(cx+r/3-18, eyeY, cx+r/3, eyeY+18), rgb(35, 35, 35), nil, 0)
		arc(img, image.Rect(cx-r/3, cy, cx+r/3, cy+r/2), 15, 165, rgb(70, 30, 30), 5)
	}
}
func renderMeme(template, text string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 900, 650))
	white := rgb(255, 255, 255)
	black := rgb(20, 20, 20)
	switch template {
	case "哭泣猫猫":
		fillRect(img, img.Bounds(), rgb(185, 218, 238))
		ellipse(img, image.Rect(230, 120, 670, 560), rgb(245, 202, 170), rgb(95, 65, 55), 8)
		fillTriangle(img, [3]image.Point{{250, 190}, {255, 50}, {370, 145}}, rgb(245, 202, 170), rgb(95, 65, 55))
		fillTriangle(img, [3]image.Point{{530, 145}, {645, 50}, {650, 190}}, rgb(245, 202, 170), rgb(95, 65, 55))
		ellipse(img, image.Rect(330, 260, 370, 300), black, nil, 0)
		ellipse(img, image.Rect(530, 260, 570, 300), black, nil, 0)
		thickLine(img, 350, 315, 300, 470, rgb(55, 160, 235), 13)
		thickLine(img, 550, 315, 600, 470, rgb(55, 160, 235), 13)
		arc(img, image.Rect(390, 330, 510, 450), 200, 340, rgb(75, 35, 35), 8)
		memeText(img, text, 12, 38, 2, white)
	case "完美":
		fillRect(img, img.Bounds(), rgb(52, 158, 76))
		for _, p := range []image.Point{{90, 100}, {790, 120}, {135, 500}, {760, 520}} {
			drawTextSize(img, "★", p.X, p.Y+52, rgb(255, 224, 75), 52)
		}
		ellipse(img, image.Rect(260, 125, 640, 505), rgb(255, 223, 80), rgb(255, 245, 170), 8)
		arc(img, image.Rect(330, 225, 570, 430), 15, 165, rgb(55, 95, 30), 18)
		ellipse(img, image.Rect(350, 235, 380, 270), rgb(55, 95, 30), nil, 0)
		ellipse(img, image.Rect(520, 235, 550, 270), rgb(55, 95, 30), nil, 0)
		memeText(img, text, 15, 38, 2, white)
	case "神图":
		fillRect(img, img.Bounds(), rgb(102, 30, 145))
		for i := 0; i < 8; i++ {
			thickLine(img, 450, 300, 60+i*110, 40, rgb(255, 215, 75), 5)
			thickLine(img, 450, 300, 60+i*110, 610, rgb(255, 120, 180), 5)
		}
		memeFace(img, 450, 315, 150, false)
		drawTextSize(img, "这不是普通的图", 185, 528, rgb(255, 235, 110), 28)
		memeText(img, text, 20, 38, 2, white)
	case "PornHub标题图":
		fillRect(img, img.Bounds(), rgb(0, 0, 0))
		fillRect(img, image.Rect(0, 225, 900, 425), rgb(255, 145, 0))
		memeText(img, "PornHub", 70, 86, 3, white)
		memeText(img, text, 270, 38, 1, rgb(0, 0, 0))
	case "举牌小人":
		fillRect(img, img.Bounds(), rgb(155, 205, 235))
		ellipse(img, image.Rect(385, 375, 515, 505), rgb(245, 203, 157), rgb(50, 40, 40), 5)
		thickLine(img, 450, 505, 450, 650, rgb(35, 35, 35), 18)
		thickLine(img, 450, 535, 315, 620, rgb(35, 35, 35), 15)
		thickLine(img, 450, 535, 585, 620, rgb(35, 35, 35), 15)
		borderedRect(img, image.Rect(105, 95, 795, 370), rgb(255, 250, 220), rgb(35, 35, 35), 8)
		memeText(img, text, 145, 38, 0, rgb(35, 35, 35))
	case "熊猫人":
		fillRect(img, img.Bounds(), rgb(245, 245, 245))
		ellipse(img, image.Rect(275, 115, 625, 490), white, black, 8)
		for _, box := range []image.Rectangle{image.Rect(270, 90, 350, 180), image.Rect(550, 90, 630, 180), image.Rect(330, 235, 430, 335), image.Rect(470, 235, 570, 335), image.Rect(390, 310, 510, 395)} {
			ellipse(img, box, black, nil, 0)
		}
		memeText(img, text, 12, 38, 0, black)
	case "狗头":
		fillRect(img, img.Bounds(), rgb(225, 190, 120))
		ellipse(img, image.Rect(240, 135, 660, 550), rgb(195, 125, 65), rgb(80, 45, 25), 8)
		fillTriangle(img, [3]image.Point{{255, 220}, {185, 55}, {355, 150}}, rgb(195, 125, 65), rgb(80, 45, 25))
		fillTriangle(img, [3]image.Point{{645, 220}, {715, 55}, {545, 150}}, rgb(195, 125, 65), rgb(80, 45, 25))
		memeFace(img, 450, 330, 130, false)
		memeText(img, text, 15, 38, 2, white)
	case "震惊":
		fillRect(img, img.Bounds(), rgb(255, 215, 95))
		memeFace(img, 450, 330, 175, true)
		memeText(img, text, 20, 38, 1, rgb(100, 20, 20))
	case "彩虹":
		colors := []color.RGBA{rgb(240, 70, 70), rgb(245, 150, 50), rgb(250, 225, 80), rgb(80, 190, 100), rgb(75, 160, 230), rgb(150, 100, 210)}
		for i, c := range colors {
			fillRect(img, image.Rect(0, i*650/6, 900, (i+1)*650/6), c)
		}
		memeText(img, text, -1, 52, 3, white)
	default:
		fillRect(img, img.Bounds(), rgb(45, 55, 100))
		borderedRect(img, image.Rect(70, 80, 830, 570), nil, rgb(255, 210, 80), 8)
		drawTextSize(img, "今日份梗图", 100, 133, rgb(255, 210, 80), 28)
		memeText(img, text, -1, 52, 2, white)
	}
	return img
}
