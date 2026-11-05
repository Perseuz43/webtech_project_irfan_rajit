package handlers

import (
    "bytes"
    "errors"
    "image"
    "image/color"
    "image/jpeg"
    "net/http"
)

const (
    maxPhotoBytes  = 3 << 20  // 3 MB upload limit
    maxPhotoPixels = 16000000 // 16 megapixels: protects the server from huge images
    photoMaxSide   = 480      // stored photos are scaled down to this many pixels
)

// processPhoto checks an uploaded picture and returns it as a small JPEG.
// The second value is a message for the user when something is wrong.
func processPhoto(data []byte) ([]byte, string) {
    switch http.DetectContentType(data) {
    case "image/png", "image/jpeg", "image/gif":
    default:
        return nil, "Only PNG, JPEG or GIF pictures are allowed"
    }
    cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
    if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
        return nil, "That file is not a valid picture"
    }
    if cfg.Width*cfg.Height > maxPhotoPixels {
        return nil, "The picture is too large (max 16 megapixels)"
    }
    img, _, err := image.Decode(bytes.NewReader(data))
    if err != nil {
        return nil, "That file is not a valid picture"
    }

    w, h := cfg.Width, cfg.Height
    if longest := max(w, h); longest > photoMaxSide {
        w = max(1, w*photoMaxSide/longest)
        h = max(1, h*photoMaxSide/longest)
    }
    small, err := scaleOnWhite(img, w, h)
    if err != nil {
        return nil, "The picture could not be processed"
    }

    var buf bytes.Buffer
    if err := jpeg.Encode(&buf, small, &jpeg.Options{Quality: 85}); err != nil {
        return nil, "The picture could not be processed"
    }
    return buf.Bytes(), ""
}

// scaleOnWhite shrinks an image to dw x dh by averaging blocks of pixels (smooth results)
// and puts it on a white background, so transparent PNGs do not turn black.
func scaleOnWhite(src image.Image, dw, dh int) (*image.RGBA, error) {
    b := src.Bounds()
    sw, sh := b.Dx(), b.Dy()
    if sw == 0 || sh == 0 || dw == 0 || dh == 0 {
        return nil, errors.New("empty image")
    }
    dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

    for y := 0; y < dh; y++ {
        sy0, sy1 := y*sh/dh, (y+1)*sh/dh
        if sy1 <= sy0 {
            sy1 = sy0 + 1
        }
        for x := 0; x < dw; x++ {
            sx0, sx1 := x*sw/dw, (x+1)*sw/dw
            if sx1 <= sx0 {
                sx1 = sx0 + 1
            }
            var r, g, bl, a, n uint64
            for sy := sy0; sy < sy1; sy++ {
                for sx := sx0; sx < sx1; sx++ {
                    cr, cg, cb, ca := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
                    r += uint64(cr)
                    g += uint64(cg)
                    bl += uint64(cb)
                    a += uint64(ca)
                    n++
                }
            }
            // the colours are premultiplied by alpha, so the missing part is white
            back := 65535 - a/n
            dst.SetRGBA(x, y, color.RGBA{
                R: uint8(min(65535, r/n+back) >> 8),
                G: uint8(min(65535, g/n+back) >> 8),
                B: uint8(min(65535, bl/n+back) >> 8),
                A: 255,
            })
        }
    }
    return dst, nil
}
