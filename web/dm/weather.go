package dm

import (
	"math/rand"
	"net/url"
	"strings"
	"time"
)

// Lobby weather: the storm between the title art and the lobby's text and
// panels. Fog and wind gusts are pure CSS (weather_wasm.go); lightning is
// scheduled here so every strike differs in timing, shape, place and size.

// boltPaths are four pre-generated forked bolts (midpoint displacement with
// two or three branches) in a -100..300 by 0..600 viewBox. Each strike picks
// one and mirrors, scales and places it, so repeats are hard to spot.
var boltPaths = [4][]string{
	{
		"M100 0 L98 10 L97 18 L98 26 L100 34 L97 43 L88 52 L88 61 L85 70 L82 81 L86 90 L85 101 L85 112 L96 121 L100 132 L105 141 L106 150 L103 160 L101 170 L101 178 L103 186 L107 196 L110 206 L117 217 L119 227 L120 237 L125 246 L124 255 L117 263 L110 272 L107 280 L102 287 L96 295 L97 305 L95 313 L95 323 L103 333 L97 343 L93 352 L98 361 L95 371 L87 380 L85 391 L85 402 L86 411 L79 421 L66 432 L63 444 L54 455 L56 465 L57 473 L58 482 L58 492 L59 501 L62 508 L62 518 L61 527 L62 535 L61 543 L57 553 L59 563 L63 573 L68 582 L75 590 L79 600",
		"M82 81 L82 88 L87 95 L95 101 L97 109 L101 116 L110 125 L118 132 L120 140 L121 146 L126 154 L128 163 L124 172 L131 179 L142 187 L149 195 L148 204",
		"M110 206 L99 220 L96 233 L94 245 L89 257 L82 270 L77 281 L68 292 L63 302 L57 313 L44 323 L40 334 L36 345 L37 356 L47 367 L45 378 L39 388",
	},
	{
		"M100 0 L101 8 L105 18 L106 29 L102 38 L103 49 L97 59 L101 66 L104 75 L96 84 L92 95 L91 106 L91 117 L97 127 L100 137 L102 147 L101 158 L102 168 L107 179 L105 191 L109 202 L104 213 L100 223 L100 231 L94 241 L93 251 L93 263 L94 272 L94 281 L94 289 L97 296 L104 305 L106 315 L102 323 L94 332 L90 340 L83 350 L79 360 L78 367 L79 376 L75 385 L79 393 L80 402 L87 408 L88 416 L83 424 L82 434 L79 441 L82 450 L83 459 L85 469 L90 477 L90 485 L90 495 L87 505 L87 514 L92 522 L99 532 L104 542 L102 550 L105 559 L107 569 L106 579 L104 590 L96 600",
		"M79 393 L68 401 L49 408 L38 417 L35 426 L17 433 L4 439 L-3 447 L-18 457 L-29 467 L-41 474 L-48 482 L-57 489 L-62 496 L-64 505 L-67 512 L-71 522",
		"M91 117 L101 130 L105 141 L107 153 L111 164 L115 175 L121 185 L122 195 L116 204 L123 215 L138 225 L153 236 L158 246 L170 257 L179 267 L184 279 L191 290",
		"M97 127 L92 133 L80 141 L65 148 L57 154 L52 162 L50 167 L47 171 L43 177 L47 181 L45 186 L41 189 L42 193 L43 197 L52 202 L54 207 L47 213",
	},
	{
		"M100 0 L98 9 L97 20 L103 30 L102 38 L104 48 L101 57 L107 66 L111 73 L107 83 L103 93 L98 104 L91 114 L91 122 L93 131 L95 141 L94 150 L92 162 L95 173 L95 183 L102 193 L106 203 L112 213 L118 224 L126 233 L127 243 L130 252 L128 263 L126 274 L127 284 L134 295 L131 305 L134 316 L134 324 L127 334 L126 343 L120 351 L118 360 L123 369 L122 378 L116 387 L114 393 L113 402 L113 410 L107 418 L103 426 L103 436 L101 444 L93 453 L89 464 L89 475 L90 486 L92 495 L93 503 L90 513 L94 522 L98 531 L97 540 L100 548 L100 558 L104 567 L112 575 L114 584 L114 592 L112 600",
		"M118 224 L124 229 L121 235 L117 239 L114 245 L115 251 L118 259 L116 265 L118 271 L124 275 L132 279 L129 283 L129 289 L130 292 L138 296 L157 301 L166 306",
		"M127 284 L124 290 L120 297 L121 304 L130 310 L126 317 L122 322 L119 327 L120 333 L117 339 L113 346 L108 350 L102 355 L98 360 L92 363 L89 367 L81 371",
	},
	{
		"M100 0 L92 9 L85 19 L81 29 L81 39 L82 49 L82 58 L78 70 L82 79 L75 88 L71 98 L67 107 L70 114 L66 124 L60 133 L58 142 L48 151 L46 160 L50 171 L48 180 L50 190 L46 198 L44 206 L44 215 L46 225 L47 233 L46 241 L43 249 L36 257 L37 267 L36 276 L39 285 L39 292 L40 302 L44 311 L40 319 L38 327 L38 336 L42 345 L44 355 L48 365 L55 374 L61 385 L68 393 L81 402 L85 411 L91 422 L93 433 L96 441 L101 451 L106 461 L108 470 L109 480 L111 492 L106 503 L104 514 L105 525 L107 535 L108 545 L102 555 L101 565 L104 573 L106 583 L108 591 L111 600",
		"M43 249 L40 258 L45 268 L42 276 L38 285 L38 296 L43 307 L41 317 L35 329 L27 341 L19 354 L6 366 L-2 380 L-10 388 L-13 399 L-19 410 L-27 422",
		"M71 98 L76 106 L74 112 L72 118 L75 127 L81 132 L88 137 L97 143 L103 148 L98 155 L97 163 L96 169 L96 176 L103 183 L102 190 L112 198 L115 206",
		"M67 107 L59 113 L52 118 L44 126 L40 132 L38 141 L27 147 L21 155 L14 163 L4 171 L-7 179 L-30 187 L-45 194 L-57 205 L-63 213 L-76 220 L-81 229",
	},
}

// boltDataURI renders bolt variant i as an SVG data URI: a wide blue glow, a
// narrower pale halo and a white-hot core, so it reads as light, not a line.
func boltDataURI(i int) string {
	paths := boltPaths[((i%len(boltPaths))+len(boltPaths))%len(boltPaths)]
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="-100 0 400 600" fill="none" stroke-linecap="round" stroke-linejoin="round">`)
	b.WriteString(`<defs><filter id="g" x="-50%" y="-10%" width="200%" height="120%"><feGaussianBlur stdDeviation="7"/></filter></defs>`)
	layer := func(stroke, width, extra string) {
		for _, part := range []string{`<g stroke="`, stroke, `" stroke-width="`, width, `"`, extra, `>`} {
			b.WriteString(part)
		}
		for n, d := range paths {
			w := ""
			if n > 0 {
				w = ` stroke-opacity=".7"`
			}
			for _, part := range []string{`<path d="`, d, `"`, w, `/>`} {
				b.WriteString(part)
			}
		}
		b.WriteString(`</g>`)
	}
	layer("#6fa8ff", "18", ` filter="url(#g)" opacity=".9"`)
	layer("#cfe6ff", "6.5", ` opacity=".95"`)
	layer("#ffffff", "2.6", "")
	b.WriteString(`</svg>`)
	return "data:image/svg+xml," + url.PathEscape(b.String())
}

// strike is one lightning event.
type strike struct {
	Delay   time.Duration // wait before this strike
	Variant int           // which boltPaths entry
	LeftPct float64       // bolt position, percent of the screen width
	TopPct  float64       // where the bolt starts, percent of the screen height
	Scale   float64       // bolt height multiplier
	Flip    bool          // mirror horizontally
	Double  bool          // a second flicker right after the first
	Burst   int           // strikes in this volley, 1 to 3
}

// nextStrike picks the next strike. Volleys are 3 to 9 seconds apart; about a
// third are bursts of two or three strikes (see burstGap). Bolts land in the
// open sky on the left two-thirds of the frame, where the sky mask is opaque.
func nextStrike(r *rand.Rand) strike {
	return strike{
		Delay:   3*time.Second + time.Duration(r.Int63n(int64(6*time.Second))),
		Variant: r.Intn(len(boltPaths)),
		LeftPct: 4 + r.Float64()*44,
		TopPct:  -6 + r.Float64()*8,
		Scale:   0.75 + r.Float64()*0.45,
		Flip:    r.Intn(2) == 0,
		Double:  r.Intn(10) < 4,
		Burst:   burstSize(r.Intn(10)),
	}
}

// burstSize turns a 0..9 roll into a volley size: 1 (70%), 2 (20%) or 3 (10%).
func burstSize(roll int) int {
	switch {
	case roll >= 9:
		return 3
	case roll >= 7:
		return 2
	}
	return 1
}

// burstGap is the pause between strikes inside a volley. At 600 ms or more,
// even a double-flicker strike stays under three flashes a second, the WCAG
// 2.3.1 photosensitivity threshold.
func burstGap(r *rand.Rand) time.Duration {
	return 600*time.Millisecond + time.Duration(r.Int63n(int64(900*time.Millisecond)))
}
