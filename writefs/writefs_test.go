package writefs

import (
	"fmt"
	"os"
	"testing"
)

// go test -run TestWrapIsNotExistErr -v ./writefs
func TestWrapIsNotExistErr(t *testing.T) {
	osErrNotExist := IsNotExist(os.ErrNotExist)
	if !osErrNotExist {
		t.Fatalf("expect %s = %+v, actual:%+v", `osErrNotExist`, true, osErrNotExist)
	}

	wrapErrNotExist := fmt.Errorf("test wrap: %w", os.ErrNotExist)
	wrapErrNotExistTest := IsNotExist(wrapErrNotExist)
	if !wrapErrNotExistTest {
		t.Fatalf("expect %s = %+v, actual:%+v", `wrapErrNotExistTest`, true, wrapErrNotExistTest)
	}

	wrapErrNotExist2 := fmt.Errorf("test wrap2: %w", wrapErrNotExist)
	wrapErrNotExistTest2 := IsNotExist(wrapErrNotExist2)
	if !wrapErrNotExistTest2 {
		t.Fatalf("expect %s = %+v, actual:%+v", `wrapErrNotExistTest2`, true, wrapErrNotExistTest2)
	}
}
