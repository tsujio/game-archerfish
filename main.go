package main

import (
	"embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	logging "github.com/tsujio/game-logging-server/client"
	"github.com/tsujio/game-util/drawutil"
	"github.com/tsujio/game-util/resourceutil"
	"github.com/tsujio/game-util/touchutil"
)

const (
	gameName             = "archerfish"
	screenWidth          = 640
	screenHeight         = 480
	fishPosXInCamera     = 0
	fishPosYInCamera     = 0
	fishPosZInCamera     = cameraF
	touchableR           = 50
	cameraF              = 50
	cameraHeight         = 120
	gravity              = 0.5
	bulletR              = 10
	normalEnemyR         = 40
	dizzyEnemyR          = 35
	shyEnemyR            = 30
	normalEnemyYInScreen = screenHeight/2 - 50.0
	dizzyEnemyYInScreen  = normalEnemyYInScreen - 70
	shyEnemyYInScreen    = dizzyEnemyYInScreen - 70
	enemyZ               = 200.0
	finishTimeInTicks    = 60 * 60
)

//go:embed resources/*.ttf resources/*.dat resources/bgm-*.wav resources/secret
var resources embed.FS

var (
	fontL, fontM, fontS = resourceutil.ForceLoadFont(resources, "resources/PressStart2P-Regular.ttf", nil)
	audioContext        = audio.NewContext(48000)
	gameStartAudioData  = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 システム49.mp3.dat", audioContext)
	timeStartAudioData  = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 笛01.mp3.dat", audioContext)
	splashAudioData     = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂  水02.mp3.dat", audioContext)
	shootAudioData      = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 点火01.mp3.dat", audioContext)
	hitAudioData        = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 システム16.mp3.dat", audioContext)
	gameOverAudioData   = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 物音15.mp3.dat", audioContext)
	rankingAudioData    = resourceutil.ForceLoadDecodedAudio(resources, "resources/魔王魂 効果音 システム46.mp3.dat", audioContext)
	bgmPlayer           = resourceutil.ForceCreateBGMPlayer(resources, "resources/bgm-archerfish.wav", audioContext)
)

func toScreenPosition(xInCamera, yInCamera, zInCamera float64) (float64, float64) {
	x := xInCamera * cameraF / zInCamera
	y := (yInCamera + cameraHeight) * cameraF / zInCamera
	return x + float64(screenWidth)/2, y + float64(screenHeight)/2
}

func toCameraPosition(xInScreen, yInScreen, zInCamera float64) (float64, float64) {
	x := (xInScreen - float64(screenWidth)/2) * zInCamera / cameraF
	y := (yInScreen-float64(screenHeight)/2)*zInCamera/cameraF - cameraHeight
	return x, y
}

type Fish struct {
	game    *Game
	ticks   uint64
	x, y, z float64
}

func (f *Fish) Update() {
	f.ticks++
}

var fishPattern1 = [][]int{
	{0, 0, 0, 0, 3, 0, 0, 0, 0},
	{0, 0, 0, 3, 3, 3, 0, 0, 0},
	{0, 0, 2, 3, 3, 3, 2, 0, 0},
	{0, 2, 1, 2, 3, 2, 1, 2, 0},
	{0, 2, 2, 2, 3, 2, 2, 2, 0},
	{0, 3, 2, 3, 3, 3, 2, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
	{0, 3, 3, 1, 1, 1, 3, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
	{3, 3, 3, 1, 1, 1, 3, 3, 3},
}

var fishPattern2 = [][]int{
	{0, 0, 0, 0, 3, 0, 0, 0, 0},
	{0, 0, 0, 3, 3, 3, 0, 0, 0},
	{0, 0, 2, 3, 3, 3, 2, 0, 0},
	{0, 2, 1, 2, 3, 2, 1, 2, 0},
	{0, 2, 2, 2, 3, 2, 2, 2, 0},
	{0, 3, 2, 3, 3, 3, 2, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
	{0, 3, 3, 1, 1, 1, 3, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
}

var fishPattern3 = [][]int{
	{0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 3, 3, 3, 0, 0, 0},
	{0, 0, 2, 3, 3, 3, 2, 0, 0},
	{0, 2, 1, 2, 3, 2, 1, 2, 0},
	{0, 2, 2, 2, 3, 2, 2, 2, 0},
	{0, 3, 2, 3, 3, 3, 2, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
	{0, 3, 3, 1, 1, 1, 3, 3, 0},
	{0, 3, 3, 3, 1, 3, 3, 3, 0},
}

var fishImages = (func() []*ebiten.Image {
	var images []*ebiten.Image
	for _, p := range [][][]int{fishPattern1, fishPattern2, fishPattern3} {
		images = append(images, drawutil.CreatePatternImage(p, &drawutil.CreatePatternImageOption[int]{
			DotSize: 5,
			ColorMap: map[int]color.Color{
				1: color.Black,
				2: color.White,
				3: color.RGBA{0xa5, 0xf5, 0xf5, 0xff},
			},
		}))
	}
	return images
})()

func (f *Fish) Draw(screen *ebiten.Image) {
	var image *ebiten.Image
	if f.game.hold {
		image = fishImages[2]
	} else {
		image = fishImages[(f.ticks/30)%2]
	}

	x, y := toScreenPosition(f.x, f.y, f.z)
	drawutil.DrawImage(screen, image, x, y, &drawutil.DrawImageOption{
		BasePosition: drawutil.DrawImagePositionCenter,
	})
}

type Bullet struct {
	ticks      uint64
	x, y, z    float64
	vx, vy, vz float64
	r          float64
}

func (b *Bullet) Update() {
	b.ticks++

	b.vy += gravity

	b.x += b.vx
	b.y += b.vy
	b.z += b.vz
}

func (b *Bullet) Draw(screen *ebiten.Image) {
	x, y := toScreenPosition(b.x, b.y, b.z)
	ebitenutil.DrawCircle(screen, x, y, bulletR*fishPosZInCamera/b.z, color.RGBA{
		R: 0x40,
		G: 0xa0,
		B: 0xff,
		A: 0xff,
	})
}

type SplashEffect struct {
	ticks   uint64
	x, y, z float64
	vx, vy  float64
}

func (e *SplashEffect) Update() {
	e.ticks++

	e.vy += gravity

	e.x += e.vx
	e.y += e.vy
}

func (e *SplashEffect) Draw(screen *ebiten.Image) {
	if e.y < 0 {
		x, y := toScreenPosition(e.x, e.y, e.z)
		ebitenutil.DrawRect(screen, x, y, 3, 3, color.White)
	}
}

type EnemyKind int

const (
	EnemyKindNormal EnemyKind = iota
	EnemyKindDizzy
	EnemyKindShy
)

type Enemy struct {
	ticks       uint64
	kind        EnemyKind
	hit         bool
	x, y, z     float64
	vx, vy, vx0 float64
	r           float64
	game        *Game
}

func (e *Enemy) Update() {
	if e.ticks == 0 {
		e.vx0 = e.vx
	}

	e.ticks++

	if e.hit {
		e.vy += gravity
		e.y += e.vy
		return
	}

	switch e.kind {
	case EnemyKindDizzy:
		if e.ticks%60 == 0 && e.game.random.Int()%2 == 0 {
			if e.vx == 0 {
				e.vx = e.vx0
			} else {
				e.vx = 0
			}
		}
	case EnemyKindShy:
		if e.ticks == 120 {
			e.vx = 0
		} else if e.ticks == 240 {
			e.vx = e.vx0 * -1
		}
	}

	e.x += e.vx
}

var enemyPatterns = [][][]rune{
	{
		[]rune(" ####  "),
		[]rune("#.#.###"),
		[]rune("###### "),
		[]rune("#######"),
		[]rune("# ##  #"),
	},
	{
		[]rune(" ####  "),
		[]rune("#######"),
		[]rune("#.#.## "),
		[]rune("#######"),
		[]rune("#   # #"),
	},
}

var normalEnemyImages = drawutil.CreatePatternImageArray(enemyPatterns, &drawutil.CreatePatternImageOption[rune]{
	ColorMap: map[rune]color.Color{
		'#': color.Black,
		'.': color.White,
	},
})

var dizzyEnemyImages = drawutil.CreatePatternImageArray(enemyPatterns, &drawutil.CreatePatternImageOption[rune]{
	ColorMap: map[rune]color.Color{
		'#': color.RGBA{0, 0, 0xff, 0xff},
		'.': color.White,
	},
})

var shyEnemyImages = drawutil.CreatePatternImageArray(enemyPatterns, &drawutil.CreatePatternImageOption[rune]{
	ColorMap: map[rune]color.Color{
		'#': color.RGBA{0xff, 0, 0, 0xff},
		'.': color.White,
	},
})

func (e *Enemy) Draw(screen *ebiten.Image) {
	x, y := toScreenPosition(e.x, e.y, e.z)
	xr, _ := toScreenPosition(e.x-e.r, e.y, e.z)
	w, _ := normalEnemyImages[0].Size()

	scaleX := math.Abs(xr-x) * 2 / float64(w)
	scaleY := scaleX

	if e.vx > 0 || e.vx == 0 && e.vx0 > 0 {
		scaleX *= -1
	}

	if e.hit {
		scaleY *= -1
	}

	var image *ebiten.Image
	switch e.kind {
	case EnemyKindNormal:
		image = normalEnemyImages[e.ticks/30%2]
	case EnemyKindDizzy:
		image = dizzyEnemyImages[e.ticks/30%2]
	case EnemyKindShy:
		image = shyEnemyImages[e.ticks/30%2]
	}

	drawutil.DrawImage(screen, image, x, y, &drawutil.DrawImageOption{
		ScaleX:       scaleX,
		ScaleY:       scaleY,
		BasePosition: drawutil.DrawImagePositionCenter,
	})
}

type GainEffect struct {
	ticks    uint64
	x, y, y0 float64
	score    int
}

func (e *GainEffect) Update() {
	if e.ticks == 0 {
		e.y0 = e.y
	}
	e.ticks++
	e.y = e.y0 - 30*math.Sin(float64(e.ticks)/60*math.Pi)
}

func (e *GainEffect) Draw(screen *ebiten.Image) {
	t := fmt.Sprintf("%+d", e.score)
	text.Draw(screen, t, fontM.Face, int(e.x), int(e.y), color.RGBA{0xff, 0xe0, 0, 0xff})
}

type Leaf struct {
	xInScreen, yInScreen float64
	scaleX, scaleY       float64
	rotate               float64
}

var leafImage = drawutil.CreatePatternImage([][]rune{
	[]rune(" ## "),
	[]rune("####"),
	[]rune("####"),
	[]rune("### "),
	[]rune(" #  "),
}, &drawutil.CreatePatternImageOption[rune]{
	ColorMap: map[rune]color.Color{
		'#': color.RGBA{0x2c, 0xda, 0x31, 0xff},
	},
	DotSize: 3,
})

func (l *Leaf) Draw(screen *ebiten.Image) {
	drawutil.DrawImage(screen, leafImage, l.xInScreen, l.yInScreen, &drawutil.DrawImageOption{
		ScaleX:       l.scaleX,
		ScaleY:       l.scaleY,
		Rotate:       l.rotate,
		BasePosition: drawutil.DrawImagePositionCenter,
	})
}

type GameMode int

const (
	GameModeTitle GameMode = iota
	GameModePlaying
	GameModeGameOver
	GameModeRanking
)

type TouchRecord struct {
	Ticks        uint64 `json:"ticks"`
	JustTouched  bool   `json:"just_touched"`
	JustReleased bool   `json:"just_released"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
}

type Game struct {
	playerID           string
	playID             string
	fixedRandomSeed    int64
	touchContext       *touchutil.TouchContext
	touchBuffer        []TouchRecord
	random             *rand.Rand
	mode               GameMode
	ticksFromModeStart uint64
	hold               bool
	score              int
	rankingChan        <-chan []logging.GameScore
	ranking            []logging.GameScore
	fish               *Fish
	bullets            []Bullet
	splashEffects      []SplashEffect
	enemies            []Enemy
	gainEffects        []GainEffect
	leaves             []Leaf
	timeInTicks        uint64
}

func (g *Game) Update() error {
	g.touchContext.Update()

	g.ticksFromModeStart++

	// Logging touches
	if g.touchContext.IsBeingTouched() || g.touchContext.IsJustReleased() {
		pos := g.touchContext.GetTouchPosition()
		g.touchBuffer = append(g.touchBuffer, TouchRecord{
			Ticks:        g.ticksFromModeStart,
			JustTouched:  g.touchContext.IsJustTouched(),
			JustReleased: g.touchContext.IsJustReleased(),
			X:            pos.X,
			Y:            pos.Y,
		})
	}
	if len(g.touchBuffer) > 0 {
		lastTicks := g.touchBuffer[len(g.touchBuffer)-1].Ticks
		if len(g.touchBuffer) >= 60 ||
			lastTicks > g.ticksFromModeStart ||
			g.ticksFromModeStart-lastTicks > 60 {
			g.sendLog(map[string]interface{}{
				"touches": g.touchBuffer,
			})
			g.touchBuffer = nil
		}
	}

	switch g.mode {
	case GameModeTitle:
		if g.touchContext.IsJustTouched() {
			g.setNextMode(GameModePlaying)

			g.sendLog(map[string]interface{}{
				"action": "start_game",
			})

			audio.NewPlayerFromBytes(audioContext, gameStartAudioData).Play()
		}
	case GameModePlaying:
		if g.ticksFromModeStart%600 == 0 {
			g.sendLog(map[string]interface{}{
				"action": "playing",
				"ticks":  g.ticksFromModeStart,
				"score":  g.score,
			})
		}

		if g.ticksFromModeStart > 3*60 {
			if g.timeInTicks == 0 {
				audio.NewPlayerFromBytes(audioContext, timeStartAudioData).Play()

				bgmPlayer.Rewind()
				bgmPlayer.Play()
			}
			g.timeInTicks++
		}

		if g.timeInTicks > 0 && g.touchContext.IsJustTouched() {
			pos := g.touchContext.GetTouchPosition()
			touchX, touchY := float64(pos.X), float64(pos.Y)
			fishX, fishY := toScreenPosition(fishPosXInCamera, fishPosYInCamera, fishPosZInCamera)
			if math.Pow(touchX-fishX, 2)+math.Pow(touchY-fishY, 2) < math.Pow(touchableR, 2) {
				g.hold = true
			}
		}

		if g.hold && g.touchContext.IsJustReleased() {
			g.hold = false

			x, y := g.getHoldPosition()
			bullet := g.newBulletByTouchPosition(x, y)
			g.bullets = append(g.bullets, *bullet)

			audio.NewPlayerFromBytes(audioContext, shootAudioData).Play()

			for i := 0; i < 5; i++ {
				_, h := fishImages[0].Size()
				g.splashEffects = append(g.splashEffects, SplashEffect{
					x:  g.fish.x,
					y:  g.fish.y - float64(h)/2,
					z:  g.fish.z,
					vx: 5.0 * math.Cos(math.Pi*g.random.Float64()),
					vy: -10.0 * math.Sin(math.Pi*g.random.Float64()),
				})
			}
		}

		// Enemy enter
		if g.ticksFromModeStart%60 == 0 {
			for _, param := range []struct {
				Kind                  EnemyKind
				AppearanceProbability float64
				Vx                    float64
			}{
				{
					Kind:                  EnemyKindNormal,
					AppearanceProbability: 0.25,
					Vx:                    2.0,
				},
				{
					Kind:                  EnemyKindDizzy,
					AppearanceProbability: 0.20,
					Vx:                    4.0,
				},
				{
					Kind:                  EnemyKindShy,
					AppearanceProbability: 0.10,
					Vx:                    4.0,
				},
			} {
				if g.random.Float64() < param.AppearanceProbability {
					var xInScreen float64
					if g.random.Int()%2 == 0 {
						xInScreen = -50
					} else {
						xInScreen = screenWidth + 50
					}

					vx := param.Vx
					if xInScreen > 0 {
						vx *= -1
					}

					var yInScreen, enemyR float64
					switch param.Kind {
					case EnemyKindNormal:
						yInScreen = normalEnemyYInScreen
						enemyR = normalEnemyR
					case EnemyKindDizzy:
						yInScreen = dizzyEnemyYInScreen
						enemyR = dizzyEnemyR
					case EnemyKindShy:
						yInScreen = shyEnemyYInScreen
						enemyR = shyEnemyR
					}

					x, y := toCameraPosition(xInScreen, yInScreen, enemyZ)

					g.enemies = append(g.enemies, Enemy{
						kind: param.Kind,
						x:    x,
						y:    y,
						z:    enemyZ,
						vx:   vx,
						r:    enemyR,
						game: g,
					})
				}
			}
		}

		// Fish
		g.fish.Update()

		// Bullets
		var newBullets []Bullet
		for i := range g.bullets {
			bullet := &g.bullets[i]

			bullet.Update()

			if bullet.y > 0 {
				for i := 0; i < 5; i++ {
					r := 10.0
					g.splashEffects = append(g.splashEffects, SplashEffect{
						x:  bullet.x,
						y:  0,
						z:  bullet.z,
						vx: r * math.Cos(math.Pi*g.random.Float64()),
						vy: -r * math.Sin(math.Pi*g.random.Float64()),
					})
				}
			} else {
				newBullets = append(newBullets, *bullet)
			}
		}
		g.bullets = newBullets

		// SplashEffects
		var newSplashEffects []SplashEffect
		for i := range g.splashEffects {
			effect := &g.splashEffects[i]

			effect.Update()

			if effect.y <= 100 {
				newSplashEffects = append(newSplashEffects, *effect)
			}
		}
		g.splashEffects = newSplashEffects

		// Enemies
		var newEnemies []Enemy
		for i := range g.enemies {
			enemy := &g.enemies[i]

			enemy.Update()

			if enemy.y > 0 {
				for i := 0; i < 5; i++ {
					r := 15.0
					g.splashEffects = append(g.splashEffects, SplashEffect{
						x:  enemy.x,
						y:  0,
						z:  enemy.z,
						vx: r * math.Cos(math.Pi*g.random.Float64()),
						vy: -r * math.Sin(math.Pi*g.random.Float64()),
					})
				}

				audio.NewPlayerFromBytes(audioContext, splashAudioData).Play()

				continue
			}

			x, _ := toScreenPosition(enemy.x, enemy.y, enemy.z)
			if x > -50 || x < screenWidth+50 {
				newEnemies = append(newEnemies, *enemy)
			}
		}
		g.enemies = newEnemies

		// Gain effects
		var newGainEffects []GainEffect
		for i := range g.gainEffects {
			effect := &g.gainEffects[i]
			effect.Update()
			if effect.ticks < 60 {
				newGainEffects = append(newGainEffects, *effect)
			}
		}
		g.gainEffects = newGainEffects

		// Bullet and enemy collision
		newBullets = nil
		for i := range g.bullets {
			b := &g.bullets[i]
			hit := false
			for j := range g.enemies {
				e := &g.enemies[j]
				if !e.hit && math.Pow(e.x-b.x, 2)+math.Pow(e.y-b.y, 2)+math.Pow(e.z-b.z, 2) < math.Pow(e.r+b.r, 2) {
					e.hit = true
					hit = true

					var score int
					switch e.kind {
					case EnemyKindNormal:
						score = 1
					case EnemyKindDizzy:
						score = 3
					case EnemyKindShy:
						score = 5
					}
					x, y := toScreenPosition(e.x, e.y, e.z)
					g.gainEffects = append(g.gainEffects, GainEffect{
						x:     x,
						y:     y,
						score: score,
					})

					g.score += score

					audio.NewPlayerFromBytes(audioContext, hitAudioData).Play()

					break
				}
			}
			if !hit {
				newBullets = append(newBullets, *b)
			}
		}
		g.bullets = newBullets

		if g.timeInTicks >= finishTimeInTicks {
			g.sendLog(map[string]interface{}{
				"action": "game_over",
				"score":  g.score,
			})

			g.setNextMode(GameModeGameOver)

			audio.NewPlayerFromBytes(audioContext, gameOverAudioData).Play()

			ch := make(chan []logging.GameScore, 1)

			go (func(playerID string, playID string, score int, c chan<- []logging.GameScore) {
				logging.RegisterScore(gameName, playerID, playID, score)
				if ranking, err := logging.GetScoreList(gameName); err == nil {
					c <- ranking
				}
				close(c)
			})(g.playerID, g.playID, g.score, ch)

			g.rankingChan = ch
		}
	case GameModeGameOver:
		if g.ticksFromModeStart > 60 && g.touchContext.IsJustTouched() {
			select {
			case ranking := <-g.rankingChan:
				g.ranking = ranking
			default:
			}

			if len(g.ranking) > 0 {
				g.setNextMode(GameModeRanking)
				audio.NewPlayerFromBytes(audioContext, rankingAudioData).Play()
			} else {
				g.initialize()
				bgmPlayer.Pause()
			}
		}
	case GameModeRanking:
		if len(g.ranking) == 0 || g.ticksFromModeStart > 60 && g.touchContext.IsJustTouched() {
			g.initialize()
			bgmPlayer.Pause()
		}
	}

	return nil
}

func (g *Game) getHoldPosition() (float64, float64) {
	pos := g.touchContext.GetTouchPosition()
	x, y := float64(pos.X), float64(pos.Y)

	_, fishY := toScreenPosition(fishPosXInCamera, fishPosYInCamera, fishPosZInCamera)
	if x < 0 {
		x = 0
	}
	if x > screenWidth {
		x = screenWidth
	}
	if y < fishY {
		y = fishY
	}
	if y > screenHeight {
		y = screenHeight
	}

	return x, y
}

func (g *Game) newBulletByTouchPosition(touchX, touchY float64) *Bullet {
	fishX, fishY := toScreenPosition(fishPosXInCamera, fishPosYInCamera, fishPosZInCamera)

	atan2 := math.Atan2(fishY-touchY, fishX-touchX)
	d := math.Sqrt(math.Pow(fishX-touchX, 2) + math.Pow(fishY-touchY, 2))
	r := 40 * d / (screenHeight - fishY)

	return &Bullet{
		x:  fishPosXInCamera,
		y:  fishPosYInCamera,
		z:  fishPosZInCamera,
		vx: r * math.Cos(atan2),
		vy: r * math.Sin(atan2),
		vz: 3,
		r:  bulletR,
	}
}

func (g *Game) drawWaterSurface(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 0, screenHeight/2, screenWidth, screenHeight/2, color.RGBA{0x0f, 0x5d, 0xfa, 0xff})
}

func (g *Game) drawScaffold(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 0, normalEnemyYInScreen+5, screenWidth, 10, color.RGBA{0xfa, 0x68, 0x35, 0xff})
	ebitenutil.DrawRect(screen, 0, dizzyEnemyYInScreen+4, screenWidth, 10, color.RGBA{0xfa, 0x68, 0x35, 0xff})
	ebitenutil.DrawRect(screen, 0, shyEnemyYInScreen+4, 90, 10, color.RGBA{0xfa, 0x68, 0x35, 0xff})
	ebitenutil.DrawRect(screen, screenWidth-90, shyEnemyYInScreen+4, 90, 10, color.RGBA{0xfa, 0x68, 0x35, 0xff})
}

func (g *Game) drawPhrase(screen *ebiten.Image) {
	t := "Drag me!"
	text.Draw(screen, t, fontS.Face, screenWidth/2-len(t)*int(fontS.FaceOptions.Size)/2, 320, color.White)
}

func (g *Game) drawSight(screen *ebiten.Image) {
	fishX, fishY := toScreenPosition(fishPosXInCamera, fishPosYInCamera, fishPosZInCamera)
	x, y := g.getHoldPosition()
	ebitenutil.DrawLine(screen, fishX, fishY, x, y, color.White)

	bullet := g.newBulletByTouchPosition(x, y)
	t := (enemyZ - bullet.z) / bullet.vz
	xt := bullet.x + bullet.vx*t
	yt := bullet.y + bullet.vy*t + gravity/2*t*t
	zt := enemyZ
	if yt > 0 {
		t = (-bullet.vy + math.Sqrt(math.Pow(bullet.vy, 2)-4*gravity/2*bullet.y)) / gravity
		xt = bullet.x + bullet.vx*t
		yt = 0
		zt = bullet.z + bullet.vz*t
	}
	xts, yts := toScreenPosition(xt, yt, zt)
	ebitenutil.DrawCircle(screen, xts, yts, 10, color.RGBA{0, 0, 0, 0x30})
}

func (g *Game) drawTime(screen *ebiten.Image) {
	if g.mode == GameModePlaying && g.ticksFromModeStart < 3*60 {
		timeText := fmt.Sprintf("%d", int(math.Ceil(float64(3*60-g.ticksFromModeStart)/60)))
		text.Draw(screen, timeText, fontL.Face, screenWidth/2-len(timeText)*int(fontL.FaceOptions.Size)/2, 260, color.White)
	} else {
		timeText := fmt.Sprintf("%d", int(math.Ceil(float64(finishTimeInTicks-g.timeInTicks)/60)))
		text.Draw(screen, timeText, fontS.Face, screenWidth/2-len(timeText)*int(fontS.FaceOptions.Size)/2, 20, color.White)
	}
}

func (g *Game) drawScore(screen *ebiten.Image) {
	scoreText := fmt.Sprintf("SCORE %d", g.score)
	text.Draw(screen, scoreText, fontS.Face, screenWidth-(len(scoreText)+1)*int(fontS.FaceOptions.Size), 20, color.White)
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	titleText := []string{"ARCHERFISH"}
	for i, s := range titleText {
		text.Draw(screen, s, fontL.Face, screenWidth/2-len(s)*int(fontL.FaceOptions.Size)/2, 75+i*int(fontL.FaceOptions.Size*1.8), color.RGBA{0, 0, 0x50, 0xff})
	}

	usageTexts := []string{"[DRAG] Set sights on", "[RELEASE] Shoot"}
	for i, s := range usageTexts {
		text.Draw(screen, s, fontS.Face, screenWidth/2-len(s)*int(fontS.FaceOptions.Size)/2, 280+i*int(fontS.FaceOptions.Size*1.8), color.White)
	}

	creditTexts := []string{"CREATOR: NAOKI TSUJIO", "FONT: Press Start 2P by CodeMan38", "SOUND EFFECT: MaouDamashii"}
	for i, s := range creditTexts {
		text.Draw(screen, s, fontS.Face, screenWidth/2-len(s)*int(fontS.FaceOptions.Size)/2, 420+i*int(fontS.FaceOptions.Size*1.8), color.White)
	}
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	gameOverText := "GAME OVER"
	text.Draw(screen, gameOverText, fontL.Face, screenWidth/2-len(gameOverText)*int(fontL.FaceOptions.Size)/2, 185, color.White)
	scoreText := []string{"YOUR SCORE IS", fmt.Sprintf("%d!", g.score)}
	for i, s := range scoreText {
		text.Draw(screen, s, fontM.Face, screenWidth/2-len(s)*int(fontM.FaceOptions.Size)/2, 275+i*int(fontM.FaceOptions.Size*2), color.White)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xc7, 0xd7, 0xc7, 0xff})

	g.drawWaterSurface(screen)

	switch g.mode {
	case GameModeTitle:
		g.drawTitle(screen)

		g.drawScaffold(screen)

		for _, t := range []struct {
			kind      EnemyKind
			xInScreen float64
			vx        float64
		}{
			{
				kind:      EnemyKindNormal,
				xInScreen: 100,
				vx:        1,
			},
			{
				kind:      EnemyKindNormal,
				xInScreen: screenWidth - 70,
				vx:        -1,
			},
			{
				kind:      EnemyKindDizzy,
				xInScreen: 250,
				vx:        1,
			},
			{
				kind:      EnemyKindDizzy,
				xInScreen: screenWidth - 150,
				vx:        -1,
			},
			{
				kind:      EnemyKindShy,
				xInScreen: 50,
				vx:        1,
			},
		} {
			x, y := toCameraPosition(t.xInScreen, map[EnemyKind]float64{
				EnemyKindNormal: normalEnemyYInScreen,
				EnemyKindDizzy:  dizzyEnemyYInScreen,
				EnemyKindShy:    shyEnemyYInScreen,
			}[t.kind], enemyZ)
			e := &Enemy{
				kind: t.kind,
				x:    x,
				y:    y,
				z:    enemyZ,
				vx:   t.vx,
				r: map[EnemyKind]float64{
					EnemyKindNormal: normalEnemyR,
					EnemyKindDizzy:  dizzyEnemyR,
					EnemyKindShy:    shyEnemyR,
				}[t.kind],
			}
			e.Draw(screen)
		}

		for i := range g.leaves {
			g.leaves[i].Draw(screen)
		}

		g.fish.Draw(screen)
	case GameModePlaying:
		for i := range g.bullets {
			if g.bullets[i].z > enemyZ {
				g.bullets[i].Draw(screen)
			}
		}

		g.drawScaffold(screen)

		for i := range g.enemies {
			g.enemies[i].Draw(screen)
		}

		for i := range g.leaves {
			g.leaves[i].Draw(screen)
		}

		for i := range g.splashEffects {
			g.splashEffects[i].Draw(screen)
		}

		for i := range g.bullets {
			if g.bullets[i].z <= enemyZ {
				g.bullets[i].Draw(screen)
			}
		}

		for i := range g.gainEffects {
			g.gainEffects[i].Draw(screen)
		}

		g.fish.Draw(screen)

		if g.timeInTicks > 0 && !g.hold && g.score == 0 {
			g.drawPhrase(screen)
		}

		if g.hold {
			g.drawSight(screen)
		}

		g.drawTime(screen)
		g.drawScore(screen)
	case GameModeGameOver, GameModeRanking:
		for i := range g.bullets {
			if g.bullets[i].z > enemyZ {
				g.bullets[i].Draw(screen)
			}
		}

		g.drawScaffold(screen)

		for i := range g.enemies {
			g.enemies[i].Draw(screen)
		}

		for i := range g.leaves {
			g.leaves[i].Draw(screen)
		}

		for i := range g.splashEffects {
			g.splashEffects[i].Draw(screen)
		}

		for i := range g.bullets {
			if g.bullets[i].z <= enemyZ {
				g.bullets[i].Draw(screen)
			}
		}

		g.fish.Draw(screen)

		g.drawTime(screen)
		g.drawScore(screen)

		if g.mode == GameModeGameOver {
			g.drawGameOver(screen)
		} else if g.mode == GameModeRanking {
			drawutil.DrawRanking(screen, g.ranking, &drawutil.DrawRankingOption{
				TitleFont: fontL,
				BodyFont:  fontM,
				PlayerID:  g.playerID,
			})
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) sendLog(payload map[string]interface{}) {
	p := map[string]interface{}{
		"player_id": g.playerID,
		"play_id":   g.playID,
	}
	for k, v := range payload {
		p[k] = v
	}

	logging.LogAsync(gameName, p)
}

func (g *Game) setNextMode(mode GameMode) {
	g.mode = mode
	g.ticksFromModeStart = 0
}

func (g *Game) initialize() {
	var playID string
	if playIDObj, err := uuid.NewRandom(); err == nil {
		playID = playIDObj.String()
	}
	g.playID = playID

	var seed int64
	if g.fixedRandomSeed != 0 {
		seed = g.fixedRandomSeed
	} else {
		seed = time.Now().Unix()
	}

	g.sendLog(map[string]interface{}{
		"action": "initialize",
		"seed":   seed,
	})

	g.random = rand.New(rand.NewSource(seed))
	g.hold = false
	g.score = 0
	g.rankingChan = nil
	g.ranking = nil
	g.fish = &Fish{
		game: g,
		x:    fishPosXInCamera,
		y:    fishPosYInCamera,
		z:    fishPosZInCamera,
	}
	g.bullets = nil
	g.splashEffects = nil
	g.enemies = nil
	g.gainEffects = nil
	g.leaves = nil
	g.timeInTicks = 0

	for _, baseY := range []float64{normalEnemyYInScreen, dizzyEnemyYInScreen, shyEnemyYInScreen} {
		x := -50.0
		for x < screenWidth {
			x += 100 + g.random.NormFloat64()*20
			if baseY == shyEnemyYInScreen {
				if x > 90 && x < screenWidth-90 {
					continue
				}
			}
			g.leaves = append(g.leaves, Leaf{
				xInScreen: x,
				yInScreen: baseY + 13 + g.random.NormFloat64()*1,
				scaleX:    (1 + g.random.NormFloat64()*0.1) * float64(g.random.Int()%2*2-1),
				scaleY:    1 + g.random.NormFloat64()*0.1,
			})
		}
	}

	g.setNextMode(GameModeTitle)
}

func main() {
	if os.Getenv("GAME_LOGGING") == "1" {
		secret, err := resources.ReadFile("resources/secret")
		if err == nil {
			logging.Enable(string(secret))
		}
	} else {
		logging.Disable()
	}

	var randomSeed int64
	if seed, err := strconv.Atoi(os.Getenv("GAME_RAND_SEED")); err == nil {
		randomSeed = int64(seed)
	}

	playerID := os.Getenv("GAME_PLAYER_ID")
	if playerID == "" {
		if playerIDObj, err := uuid.NewRandom(); err == nil {
			playerID = playerIDObj.String()
		}
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Archerfish")

	game := &Game{
		playerID:        playerID,
		fixedRandomSeed: randomSeed,
		touchContext:    touchutil.CreateTouchContext(),
	}
	game.initialize()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
