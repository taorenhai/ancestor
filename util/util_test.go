package util

import (
	"testing"
)

func Test_IsFileExist(t *testing.T) {
	testfile1 := "/tmp/fcitx-qimpanel:0.pid"
	testfile2 := "/tmp/nodetest.pid"

	if flag := IsFileExists(testfile1); !flag {
		t.Fatalf("the result is %v, want: true", flag)
	}

	if flag := IsFileExists(testfile2); flag {
		t.Fatalf("the result is %v, wantt: false", flag)
	}

	t.Logf("test success")
}
