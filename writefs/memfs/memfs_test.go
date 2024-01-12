package memfs

import (
	"os"
	"testing"
)

// go test -run TestCreateAndList -v ./writefs/memfs
func TestCreateAndList(t *testing.T) {
	fs := NewMemFS()

	err := fs.MkdirAll("a/b/c", 0755)
	if err != nil {
		t.Fatal(err)
	}

	infos, err := fs.ReadDir("a/b/")
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("expect %s = %+v, actual:%+v", `len(infos)`, 1, len(infos))
	}

	if !infos[0].IsDir() || infos[0].Name() != "c" {
		t.Fatalf("expect read dir return isDir and name c: %v %v", infos[0].IsDir(), infos[0].Name())
	}

	w, err := fs.OpenFileWrite("a/b/c/d.txt")
	if err != nil {
		t.Fatal(err)
	}
	s := "hello world"
	_, err = w.Write([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	stat, err := fs.Stat("a/b/c/d.txt")
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != int64(len(s)) {
		t.Fatalf("expect %s = %+v, actual:%+v", `stat.Size()`, int64(len(s)), stat.Size())
	}

	r, err := fs.OpenFileRead("a/b/c/d.txt")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(s))
	_, err = r.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != s {
		t.Fatalf("expect %s = %+v, actual:%+v", `readBuf`, s, string(buf))
	}

	err = fs.RemoveAll("a/b/c")
	if err != nil {
		t.Fatal(err)
	}
	infosRm, err := fs.ReadDir("a/b")
	if err != nil {
		t.Fatal(err)
	}
	if len(infosRm) != 0 {
		t.Fatalf("expect %s = %+v, actual:%+v", `len(infosRm)`, 0, len(infosRm))
	}
}

// go test -run TestRemoveOsNonExistFile -v ./writefs/memfs
func TestRemoveOsNonExistFile(t *testing.T) {
	err := os.Remove("this_is_just_a_shit")
	if err == nil {
		t.Fatal("expect error: no such file or directory")
	}
	if err.Error() != "remove this_is_just_a_shit: no such file or directory" {
		t.Fatalf("expect error: remove this_is_just_a_shit: no such file or directory,actual: %s", err.Error())
	}
}
