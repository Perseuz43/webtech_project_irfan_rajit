package handlers

import (
    "fmt"
    "html/template"
    "math"
    "strconv"
    "strings"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

// SchoolInfo is what every page needs to show the school's name, logo and colours.
type SchoolInfo struct {
    Name        string
    Initial     string
    HasLogo     bool
    LogoVersion string
    CSS         template.CSS
}

func buildSchoolInfo(s models.School) SchoolInfo {
    info := SchoolInfo{Name: s.Name, HasLogo: s.HasLogo, LogoVersion: s.LogoVersion, Initial: "S"}
    if runes := []rune(strings.TrimSpace(s.Name)); len(runes) > 0 {
        info.Initial = strings.ToUpper(string(runes[0]))
    }
    if c, ok := parseHex(s.Accent); ok {
        info.CSS = template.CSS(accentCSS(c))
    }
    return info
}

type rgb struct{ r, g, b float64 }

var (
    white = rgb{255, 255, 255}
    black = rgb{0, 0, 0}
)

// parseHex reads "#rrggbb".
func parseHex(s string) (rgb, bool) {
    if len(s) != 7 || s[0] != '#' {
        return rgb{}, false
    }
    v, err := strconv.ParseUint(s[1:], 16, 32)
    if err != nil {
        return rgb{}, false
    }
    return rgb{float64((v >> 16) & 0xff), float64((v >> 8) & 0xff), float64(v & 0xff)}, true
}

func (c rgb) hex() string {
    f := func(v float64) int { return int(math.Round(math.Max(0, math.Min(255, v)))) }
    return fmt.Sprintf("#%02x%02x%02x", f(c.r), f(c.g), f(c.b))
}

func mix(a, b rgb, t float64) rgb {
    return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func lin(v float64) float64 {
    v /= 255
    if v <= 0.03928 {
        return v / 12.92
    }
    return math.Pow((v+0.055)/1.055, 2.4)
}

func (c rgb) lum() float64 {
    return 0.2126*lin(c.r) + 0.7152*lin(c.g) + 0.0722*lin(c.b)
}

// contrast is the WCAG contrast ratio between two colours.
func contrast(a, b rgb) float64 {
    la, lb := a.lum(), b.lum()
    if la < lb {
        la, lb = lb, la
    }
    return (la + 0.05) / (lb + 0.05)
}

// darkenUntil darkens a colour until white text on it is readable.
func darkenUntil(c rgb, want float64) rgb {
    for i := 0; i < 40 && contrast(c, white) < want; i++ {
        c = mix(c, black, 0.08)
    }
    return c
}

// lightenUntil lightens a colour until it is readable on the given background.
func lightenUntil(c, bg rgb, want float64) rgb {
    for i := 0; i < 40 && contrast(c, bg) < want; i++ {
        c = mix(c, white, 0.08)
    }
    return c
}

// accentCSS builds CSS that overrides the theme colours. It only ever contains values made
// from a validated #rrggbb colour.
func accentCSS(a rgb) string {
    surfaceDark := rgb{22, 32, 51}

    btn := darkenUntil(a, 4.5) // buttons carry white text
    btnHover := mix(btn, black, 0.15)
    nav := darkenUntil(mix(a, black, 0.35), 8)

    dHover := mix(btn, white, 0.15)
    dLink := lightenUntil(a, surfaceDark, 4.5)
    dNav := darkenUntil(mix(a, rgb{10, 16, 27}, 0.75), 8)

    light := fmt.Sprintf("--btn:%s;--btn-hover:%s;--link:%s;--nav-bg:%s",
        btn.hex(), btnHover.hex(), btn.hex(), nav.hex())
    dark := fmt.Sprintf("--btn:%s;--btn-hover:%s;--link:%s;--nav-bg:%s",
        btn.hex(), dHover.hex(), dLink.hex(), dNav.hex())

    return fmt.Sprintf(`:root{%s}
:root[data-theme="dark"]{%s}
@media (prefers-color-scheme: dark){:root:not([data-theme="light"]):not([data-theme="dark"]){%s}}
.brand-mark{background:%s}
input:focus,select:focus,textarea:focus{box-shadow:0 0 0 3px rgba(%d,%d,%d,0.28)}`,
        light, dark, dark, btn.hex(), int(btn.r), int(btn.g), int(btn.b))
}
