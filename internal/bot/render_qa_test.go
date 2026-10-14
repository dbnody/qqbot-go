package bot

import (
	"golang.org/x/image/draw"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestAllTemplateRendering(t *testing.T) {
	templates := []string{"哭泣猫猫", "完美", "神图", "PornHub标题图", "举牌小人", "熊猫人", "狗头", "震惊", "经典", "彩虹"}
	sheet := image.NewRGBA(image.Rect(0, 0, 1800, 1950))
	for i, name := range templates {
		img := renderMeme(name, name+" 今天也要努力上班")
		if img.Bounds().Dx() != 900 {
			t.Fatal(name)
		}
		rect := image.Rect((i%3)*600, (i/3)*440, (i%3)*600+600, (i/3)*440+433)
		draw.NearestNeighbor.Scale(sheet, rect, img, img.Bounds(), draw.Src, nil)
	}
	dir := os.Getenv("QQBOT_QA_DIR")
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "meme-contact.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, sheet); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}
func TestTarotOrientations(t *testing.T) {
	t.Chdir(t.TempDir())
	var upright, reversed image.Image
	for _, rev := range []bool{false, true} {
		path, err := renderTarot(tarotDeck[0], rev, "123")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if rev {
			reversed = img
		} else {
			upright = img
		}
		if dir := os.Getenv("QQBOT_QA_DIR"); dir != "" {
			name := "tarot-upright.png"
			if rev {
				name = "tarot-reversed.png"
			}
			f, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, img)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if upright.At(360, 320) == reversed.At(360, 320) {
		t.Fatal("orientation does not affect triangle")
	}
}
