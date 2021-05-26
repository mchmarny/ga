package ga

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

const (
	foo = "foo"
	bar = "bar"
)

func TestAction_IssueCommand(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.IssueCommand(&Command{
		Name:    foo,
		Message: bar,
	})

	if got, want := b.String(), "::foo::bar\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_IssueFileCommand(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile(".", ".issue_file_cmd_test_")
	if err != nil {
		t.Fatalf("unable to create a temp env file: %s", err)
	}

	defer os.Remove(file.Name())

	fakeGetenvFunc := newFakeGetenvFunc(t, "GITHUB_FOO", file.Name())
	var b bytes.Buffer
	a := NewWithWriter(&b)

	err = a.issueFileCommand(&Command{
		Name:    foo,
		Message: bar,
	}, fakeGetenvFunc)

	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// expect an empty stdout buffer
	if got, want := b.String(), ""; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	// expect the message to be written to the env file
	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Errorf("unable to read temp env file: %s", err)
	}

	if got, want := string(data), "bar\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_AddMask(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.AddMask("foobar")

	if got, want := b.String(), "::add-mask::foobar\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_AddMatcher(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.AddMatcher("foobar.json")

	if got, want := b.String(), "::add-matcher::foobar.json\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_RemoveMatcher(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.RemoveMatcher("foobar")

	if got, want := b.String(), "::remove-matcher owner=foobar::\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_AddPath(t *testing.T) {
	t.Parallel()

	const envGitHubPath = "GITHUB_PATH"

	// expect a regular command to be issued when env file is not set.
	fakeGetenvFunc := newFakeGetenvFunc(t, envGitHubPath, "")
	var b bytes.Buffer
	a := NewWithWriter(&b)

	a.addPath("/custom/bin", fakeGetenvFunc)
	if got, want := b.String(), "::add-path::/custom/bin\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	b.Reset()

	// expect a file command to be issued when env file is set.
	file, err := ioutil.TempFile(".", ".add_path_test_")
	if err != nil {
		t.Fatalf("unable to create a temp env file: %s", err)
	}

	defer os.Remove(file.Name())
	fakeGetenvFunc = newFakeGetenvFunc(t, envGitHubPath, file.Name())

	a.addPath("/custom/bin", fakeGetenvFunc)

	if got, want := b.String(), ""; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	// expect an empty stdout buffer
	if got, want := b.String(), ""; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	// expect the message to be written to the file.
	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("unable to read temp env file: %s", err)
	}

	if got, want := string(data), "/custom/bin\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_SaveState(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.SaveState("key", "value")

	if got, want := b.String(), "::save-state name=key::value\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_GetInput(t *testing.T) {
	t.Parallel()

	fakeGetenvFunc := newFakeGetenvFunc(t, "INPUT_FOO", bar)

	var b bytes.Buffer
	a := NewWithWriter(&b)
	if got, want := a.getInput("foo", fakeGetenvFunc), bar; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_GetMapInput(t *testing.T) {
	t.Parallel()

	v := `k1:v1
	      k2:v2
	      k3:v3`

	os.Setenv("INPUT_FOO", v)

	var b bytes.Buffer
	a := NewWithWriter(&b)
	m, err := a.GetMapInput("foo", ":")
	if err != nil {
		t.Errorf("error parsing input %v", err)
	}
	if len(m) != 3 {
		t.Errorf("expected 3 items, got %d", len(m))
	}
	for i := 1; i <= len(m); i++ {
		k := fmt.Sprintf("k%d", i)
		v, ok := m[k]
		if !ok {
			t.Errorf("key %s not set", k)
		}
		if v != fmt.Sprintf("v%d", i) {
			t.Errorf("invalid value, expected: v%d, got: %s", i, v)
		}
	}

	m, err = a.GetMapInput("foo", "|")
	if err == nil {
		t.Errorf("expected error, got %d items", len(m))
	}

	os.Setenv("INPUT_FOO", "")
	m, err = a.GetMapInput("foo", ":")
	if err != nil || len(m) != 0 {
		t.Errorf("expected 0 items, got %d - %v", len(m), err)
	}

	os.Unsetenv("INPUT_FOO")
	m, err = a.GetMapInput("foo", ":")
	if err != nil || len(m) != 0 {
		t.Errorf("expected 0 items, got %d - %v", len(m), err)
	}
}

func TestAction_GetSliceInput(t *testing.T) {
	t.Parallel()

	v := `item1
	item2
	item3`

	os.Setenv("INPUT_FOO", v)

	var b bytes.Buffer
	a := NewWithWriter(&b)
	items := a.GetSliceInput("foo")
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
	for i, item := range items {
		t.Logf("item[%d]: '%s'", i, item)
	}

	os.Setenv("INPUT_FOO", "")
	items = a.GetSliceInput("foo")
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d: '%s'", len(items), strings.Join(items, "\n"))
	}

	os.Unsetenv("INPUT_FOO")
	items = a.GetSliceInput("foo")
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d: '%s'", len(items), strings.Join(items, "\n"))
	}
}

func TestAction_Group(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.Group("mygroup")

	if got, want := b.String(), "::group::mygroup\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_EndGroup(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.EndGroup()

	if got, want := b.String(), "::endgroup::\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_SetEnv(t *testing.T) {
	t.Parallel()

	const envGitHubEnv = "GITHUB_ENV"

	// expectations for regular set-env commands
	checks := []struct {
		key, value, want string
	}{
		{"key", "value", "::set-env name=key::value\n"},
		{"key", "this is 100% a special\n\r value!", "::set-env name=key::this is 100%25 a special%0A%0D value!\n"},
	}

	for _, check := range checks {
		fakeGetenvFunc := newFakeGetenvFunc(t, envGitHubEnv, "")
		var b bytes.Buffer
		a := NewWithWriter(&b)
		a.setEnv(check.key, check.value, fakeGetenvFunc)
		if got, want := b.String(), check.want; got != want {
			t.Errorf("SetEnv(%q, %q): expected %q; got %q", check.key, check.value, want, got)
		}
	}

	// expectations for env file env commands
	var b bytes.Buffer
	a := NewWithWriter(&b)
	file, err := ioutil.TempFile(".", ".set_env_test_")
	if err != nil {
		t.Fatalf("unable to create a temp env file: %s", err)
	}

	defer os.Remove(file.Name())
	fakeGetenvFunc := newFakeGetenvFunc(t, envGitHubEnv, file.Name())

	a.setEnv("key", "value", fakeGetenvFunc)

	// expect an empty stdout buffer
	if got, want := b.String(), ""; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}

	// expect the command to be written to the file.
	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("unable to read temp env file: %s", err)
	}

	want := "key<<_GitHubActionsFileCommandDelimeter_\nvalue\n_GitHubActionsFileCommandDelimeter_\n"
	if got := string(data); got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_SetOutput(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.SetOutput("key", "value")

	if got, want := b.String(), "::set-output name=key::value\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_Debugf(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.Debugf("fail: %s", "thing")

	if got, want := b.String(), "::debug::fail: thing\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_Errorf(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.Errorf("fail: %s", "thing")

	if got, want := b.String(), "::error::fail: thing\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_Warningf(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.Warningf("fail: %s", "thing")

	if got, want := b.String(), "::warning::fail: thing\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_Infof(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a.Infof("info: %s\n", "thing")

	if got, want := b.String(), "info: thing\n"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

const (
	actionTestStr = "::debug file=app.js,line=100::fail: thing\n"
)

func TestAction_WithFieldsSlice(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a = a.WithFieldsSlice([]string{"line=100", "file=app.js"})
	a.Debugf("fail: %s", "thing")

	if got, want := b.String(), actionTestStr; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestAction_WithFieldsMap(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	a := NewWithWriter(&b)
	a = a.WithFieldsMap(map[string]string{"line": "100", "file": "app.js"})
	a.Debugf("fail: %s", "thing")

	if got, want := b.String(), actionTestStr; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

// newFakeGetenvFunc returns a new getenvFunc that is expected to be called with
// the provided key. It returns the provided value if the call matches the
// provided key. It reports an error on test t otherwise.
func newFakeGetenvFunc(t *testing.T, wantKey, v string) getenvFunc {
	return func(gotKey string) string {
		if gotKey != wantKey {
			t.Errorf("expected call GetenvFunc(%q) to be GetenvFunc(%q)", gotKey, wantKey)
		}

		return v
	}
}
