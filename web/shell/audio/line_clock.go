package audio

// lineClock anchors each voice line's chunks to one AudioContext base time
// and remembers when every line's scheduled audio ends. It is pure so the
// timing rules are unit-tested outside the browser; times are AudioContext
// seconds.
type lineClock struct {
	base map[string]float64
	end  map[string]float64
}

// place returns the context time a chunk starting offset seconds into its
// line should start. The first chunk anchors the line at now; a chunk that
// arrives after its slot (a network stall) shifts the rest of the line so it
// stays contiguous instead of overlapping the backlog.
func (c *lineClock) place(id string, offset, duration, now float64) float64 {
	if c.base == nil {
		c.base = make(map[string]float64)
		c.end = make(map[string]float64)
	}
	base, ok := c.base[id]
	if !ok {
		base = now
		c.base[id] = base
	}
	when := base + offset
	if when < now {
		lead := JitterLead.Seconds()
		c.base[id] = base + (now - when) + lead
		when = now + lead
	}
	if end := when + duration; end > c.end[id] {
		c.end[id] = end
	}
	return when
}

// finish forgets the line's base once its final chunk is placed; its end
// time is kept until the audio has played out.
func (c *lineClock) finish(id string) { delete(c.base, id) }

// endOf reports when the line's last scheduled chunk stops playing.
func (c *lineClock) endOf(id string) (float64, bool) {
	end, ok := c.end[id]
	return end, ok
}

// drop forgets a line entirely.
func (c *lineClock) drop(id string) {
	delete(c.base, id)
	delete(c.end, id)
}

// expired returns finished lines whose audio ended before now, so the player
// can release their sources and gain nodes. Lines still receiving chunks are
// never expired.
func (c *lineClock) expired(now float64) []string {
	var ids []string
	for id, end := range c.end {
		if _, open := c.base[id]; !open && end < now {
			ids = append(ids, id)
		}
	}
	return ids
}
