package fzf

import (
	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
	"regexp"
	"strconv"
	"strings"
)

const (
	gSixelBegin     = "\033P"
	gSixelTerminate = "\033\\"

	gEscapeCode = 27
)

type sixelScreen struct {
	wpx, hpx     int
	fontw, fonth int
}

func (sxs *sixelScreen) updateSizes(wc, hc int, window tui.Window) {
	var err error
	sxs.wpx, sxs.hpx, err = window.GetTermPixels()
	if err != nil {
		sxs.wpx, sxs.hpx = -1, -1
		sxs.fontw, sxs.fonth = -1, -1
	}

	sxs.fontw = sxs.wpx / wc
	sxs.fonth = sxs.hpx / hc
}

func (sxs *sixelScreen) pxToCells(wpx, hpx int) (int, int) {
	basew := wpx / sxs.fontw
	if wpx%sxs.fontw > 0 {
		basew += 1
	}

	baseh := hpx / sxs.fonth
	if hpx%sxs.fonth > 0 {
		baseh += 1
	}

	return basew, baseh
}

var reNumber = regexp.MustCompile(`^[0-9]+`)

// needs some testing
func sixelDimPx(s string) (w int, h int) {
	// TODO maybe take into account pixel aspect ratio

	// General sixel sequence:
	//    DCS <P1>;<P2>;<P3>;	q  [" <raster_attributes>]   <main_body> ST
	// DCS is "ESC P"
	// We are not interested in P1~P3
	// the optional raster attributes may contain the 'reported' image size in pixels
	// (The actual image can be larger, but is at least this big)
	// ST is the terminating string "ESC \"
	i := strings.Index(s, "q") + 1
	if i == 0 {
		// syntax error
		return -1, -1
	}

	// Start of (optional) Raster Attributes
	//    "	Pan	;	Pad;	Ph;	Pv
	// pixel aspect ratio = Pan/Pad
	// We are only interested in Ph and Pv (horizontal and vertical size in px)
	if s[i] == '"' {
		i++
		b := strings.Index(s[i:], ";")
		// pan := strconv.Atoi(s[a:b])
		i += b + 1
		b = strings.Index(s[i:], ";")
		// pad := strconv.Atoi(s[a:b])

		i += b + 1
		b = strings.Index(s[i:], ";")
		ph, err1 := strconv.Atoi(s[i : i+b])

		i += b + 1
		b = strings.Index(s[i:], "#")
		pv, err2 := strconv.Atoi(s[i : i+b])
		i += b

		if err1 != nil || err2 != nil {
			goto main_body // keep trying
		}

		// TODO
		// ph and pv are more like suggestions, it's still possible to go over the
		// reported size, so we might need to parse the entire main body anyway
		return ph, pv
	}

main_body:
	var wi int
	for ; i < len(s)-2; i++ {
		c := s[i]
		switch {
		case '?' <= c && c <= '~':
			wi++
		case c == '-':
			w = util.Max(w, wi)
			wi = 0
			h++
		case c == '$':
			w = util.Max(w, wi)
			wi = 0
		case c == '!':
			m := reNumber.FindString(s[i+1:])
			if m == "" {
				// syntax error
				return -1, -1
			}
			if s[i+1+len(m)] < '?' || s[i+1+len(m)] > '~' {
				// syntax error
				return -1, -1
			}
			n, _ := strconv.Atoi(m)
			wi += n - 1
		default:
		}
	}
	if s[len(s)-3] != '-' {
		w = util.Max(w, wi)
		h++ // add newline on last row
	}
	return w, h * 6
}

func (sxs *sixelScreen) sortSixelLines(lines []string) []string {
	var precessSixel = false
	var sb strings.Builder
	var newLines []string
	var sixelStartLine int
	var start int

	for index, line := range lines {
		if !precessSixel {
			start = strings.Index(line, gSixelBegin)
			if start >= 0 {
				precessSixel = true
				sixelStartLine = index
				if start > 0 {
					newLines = append(newLines, line[:start])
				}
			} else {
				newLines = append(newLines, line)
			}
		}
		if precessSixel {
			end := strings.Index(line, gSixelTerminate)

			if end > 0 {

				if index == sixelStartLine {
					sb.WriteString(line[start : end+2])
				} else {
					sb.WriteString(line[:end+2])
				}

				sixelLines := sb.String()
				sb.Reset()
				newLines = append(newLines, sixelLines)

				precessSixel = false

				if len(line) > end+2 {
					newLines = append(newLines, line[end+2:])
				}
			} else {
				if sixelStartLine != index {
					sb.WriteString(line)
				}
			}
		}
	}

	return newLines
}
