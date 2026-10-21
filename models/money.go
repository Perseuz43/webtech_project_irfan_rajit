package models

import "strconv"

// Money is a whole amount in Tk. Printing it in a template shows "Tk 1,500".
type Money int64

func (m Money) String() string {
    neg := m < 0
    if neg {
        m = -m
    }
    digits := strconv.FormatInt(int64(m), 10)
    out := make([]byte, 0, len(digits)+len(digits)/3)
    for i := 0; i < len(digits); i++ {
        if i > 0 && (len(digits)-i)%3 == 0 {
            out = append(out, ',')
        }
        out = append(out, digits[i])
    }
    s := "Tk " + string(out)
    if neg {
        s = "-" + s
    }
    return s
}
