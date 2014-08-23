package selfupdate

import (
	"fmt"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestVersion(t *testing.T) {

	v1 := Version("1")
	Convey(fmt.Sprintf("Version '%s' must be valid", v1), t, func() {
		So(v1.Valid(), ShouldBeTrue)
	})

	v1 = Version("0")
	Convey(fmt.Sprintf("Version '%s' must be valid", v1), t, func() {
		So(v1.Valid(), ShouldBeTrue)
	})

	v1 = Version("")
	Convey(fmt.Sprintf("Version '%s' must NOT be valid", v1), t, func() {
		So(v1.Valid(), ShouldBeFalse)
	})

	v1 = Version("1.0.1")
	Convey(fmt.Sprintf("Version '%s' must be valid", v1), t, func() {
		So(v1.Valid(), ShouldBeTrue)
	})

	v1 = Version("1.0.1")
	v2 := Version("1.0.1")
	Convey(fmt.Sprintf("Version '%s' must be equal to '%s'", v1, v2), t, func() {
		cond, err := v1.IsEqual(v2)
		So(err, ShouldBeNil)
		So(cond, ShouldBeTrue)
	})

	v1 = Version("1.0.1.0")
	v2 = Version("1.0.1")
	Convey(fmt.Sprintf("Version '%s' must be equal to '%s'", v1, v2), t, func() {
		cond, err := v1.IsEqual(v2)
		So(err, ShouldBeNil)
		So(cond, ShouldBeTrue)
	})

	v1 = Version("1.0.1.0")
	v2 = Version("1.0.1.1")
	Convey(fmt.Sprintf("Version '%s' must NOT be equal to '%s'", v1, v2), t, func() {
		cond, err := v1.IsEqual(v2)
		So(err, ShouldBeNil)
		So(cond, ShouldBeFalse)
	})
}
