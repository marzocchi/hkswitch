package systemd

import (
	"fmt"
	"github.com/andreyvit/diff"
	"io/ioutil"
	"strings"
	"testing"
)

func TestUnit_Write(t *testing.T) {
	u := Unit{
		WorkingDir:  "/var/run",
		Description: "test",
		CommandLine: "cmd arg",
		Command:     []string{"cmd", "arg"},
		Home:        "/home/user",
		Env: []EnvVar{
			{
				Name:  "VARNAME1",
				Value: "VARVALUE1",
			},
			{
				Name:  "VARNAME2",
				Value: "VARVALUE2",
			},
		},
	}

	testCases := []struct {
		initType string
	}{
		{
			initType: "launchd",
		},
		{
			initType: "systemd",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s", testCase.initType), func(t *testing.T) {
			buf := &strings.Builder{}
			if err := u.Write(buf, testCase.initType); err != nil {
				t.Fatal(err)
			}

			data, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.golden", testCase.initType))
			if err != nil {
				t.Fatal(err)
			}

			if is, want := strings.TrimSpace(buf.String()), strings.TrimSpace(string(data)); is != want {
				t.Fatalf("is =\n%s\n, want =\n%s\n, diff =\n%s\n", is, want, diff.LineDiff(is, want))
			}
		})
	}

}
