package selfupdate

import (
	"errors"
	"strconv"
	"strings"
)

/****************************************************************
** Version
********/

var (
	ErrVersionNegative = errors.New("Negative version number")
	ErrVersionEmpty    = errors.New("Version is empty")
)

type Version string

func (this Version) parse() ([]int, error) {
	partsStr := strings.Split(string(this), ".")
	parts := []int{}
	for _, partStr := range partsStr {
		if len(partStr) == 0 {
			continue
		}
		part, err := strconv.ParseInt(partStr, 10, 32)
		if err != nil {
			return parts, err
		}
		if part < 0 {
			return parts, ErrVersionNegative
		}
		parts = append(parts, int(part))
	}
	if len(parts) == 0 {
		return parts, ErrVersionEmpty
	}
	return parts, nil
}

func (this Version) compare(another Version) (res int, err error) {
	vs1, err := this.parse()
	if err != nil {
		return
	}
	vs2, err := another.parse()
	if err != nil {
		return
	}

	for c := 0; c < len(vs1) || c < len(vs2); c++ {
		v1 := 0
		v2 := 0
		if c < len(vs1) {
			v1 = vs1[c]
		}
		if c < len(vs2) {
			v2 = vs2[c]
		}
		if v1 > v2 {
			return +1, nil
		} else if v1 < v2 {
			return -1, nil
		}
	}

	return 0, nil
}

func (this Version) IsGreater(another Version) (bool, error) {
	cmp, err := this.compare(another)
	return cmp == 1, err
}

func (this Version) IsLesser(another Version) (bool, error) {
	cmp, err := this.compare(another)
	return cmp == -1, err
}

func (this Version) IsEqual(another Version) (bool, error) {
	cmp, err := this.compare(another)
	return cmp == 0, err
}

func (this Version) Valid() bool {
	_, err := this.parse()
	if err != nil {
		return false
	}
	return true
}
